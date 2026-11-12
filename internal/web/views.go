package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

var errInvalidFilter = errors.New("invalid filter")

var statuses = []enums.TaskStatus{enums.TaskStatusTodo, enums.TaskStatusInProgress, enums.TaskStatusReview, enums.TaskStatusDone, enums.TaskStatusCancelled}

func statusClass(status enums.TaskStatus) string {
	return strings.ReplaceAll(string(status), "_", "-")
}

type Option struct {
	Value    string
	Label    string
	Selected bool
}

type Select struct {
	Name     string
	Label    string
	Value    string
	Options  []Option
	Multiple bool
}

type Person struct {
	Name     string
	Initials string
	Username string
}

type Card struct {
	models.Task
	Assignee    Person
	CodeLinks   []models.CodeLink
	CodeChanges models.CodeSummary
}

type Column struct {
	Status enums.TaskStatus
	Tasks  []Card
}

type BoardPage struct {
	Key      string
	Name     string
	Subtitle string
	Selects  []Select
	Columns  []Column
	Count    int
	HasBoard bool
	Stats    BoardStats
	Summary  []StatusCount
	InSprint bool
}

type BoardStats struct {
	Total        int
	Done         int
	Open         int
	Cancelled    int
	Completion   int
	ChangedLines int64
	CodeChanges  models.CodeSummary
}

type StatusCount struct {
	Status enums.TaskStatus
	Count  int
}

type RelatedTask struct {
	ID     string
	Title  string
	Label  string
	Status enums.TaskStatus
}

type Comment struct {
	models.Comment
	Author Person
}

type TaskPage struct {
	models.Task
	CodeLinks   []models.CodeLink
	CodeChanges models.CodeSummary
	BoardName   string
	Assignee    Person
	Creator     Person
	Sprint      string
	Related     []RelatedTask
	Comments    []Comment
}

func (a *Application) boardPage(r *http.Request) (*Page, error) {
	ctx := r.Context()
	var boards []models.Board
	for offset := 0; ; offset += 200 {
		batch, err := a.store.ListBoards(ctx, nil, 200, offset)
		if err != nil {
			return nil, err
		}
		boards = append(boards, batch...)
		if len(batch) < 200 {
			break
		}
	}
	users, err := a.store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	query := r.URL.Query()
	key := query.Get("board")
	explicitBoard := query.Has("board")
	if r.URL.Path != "/" {
		key, explicitBoard = strings.TrimPrefix(r.URL.Path, "/"), true
	}
	if !explicitBoard && len(boards) > 0 {
		key = boards[0].Key
	}
	view := &BoardPage{Key: key, Name: "Все задачи", Subtitle: "Все доски · Все спринты", HasBoard: len(boards) > 0}
	boardOptions := []Option{{Value: "all", Label: "Все доски", Selected: key == "all" || key == ""}}
	found := key == "all" || key == ""
	filter := store.TaskFilter{Sort: "created_at", Direction: "desc", Limit: 500}
	for _, board := range boards {
		boardOptions = append(boardOptions, Option{Value: board.Key, Label: board.Key + " · " + board.Name, Selected: board.Key == key})
		if board.Key == key {
			found, view.Name = true, board.Name
			filter.BoardKey = &key
			view.Subtitle = "Все спринты"
		}
	}
	if !found {
		return nil, store.ErrBoardNotFound
	}
	view.Selects = append(view.Selects, makeSelect("board", "Доска", boardOptions))
	people := peopleMap(users)
	assignee := query.Get("assignee")
	userOptions := []Option{{Value: "", Label: "Все исполнители", Selected: assignee == ""}, {Value: "unassigned", Label: "Без исполнителя", Selected: assignee == "unassigned"}}
	if assignee == "unassigned" {
		filter.Unassigned = true
	} else if assignee != "" {
		var id uuid.UUID
		if err = id.UnmarshalText([]byte(assignee)); err != nil {
			return nil, errInvalidFilter
		}
		if _, ok := people[id.String()]; !ok {
			return nil, errInvalidFilter
		}
		assignee = id.String()
		filter.AssigneeID = &id
	}
	for _, user := range users {
		userOptions = append(userOptions, Option{Value: user.ID.String(), Label: personName(user), Selected: user.ID.String() == assignee})
	}
	view.Selects = append(view.Selects, makeSelect("assignee", "Исполнитель", userOptions))
	sprint := query.Get("sprint")
	sprintOptions := []Option{{Value: "", Label: "Все спринты", Selected: sprint == ""}, {Value: "backlog", Label: "Без спринта", Selected: sprint == "backlog"}}
	if filter.BoardKey != nil {
		var sprints []models.Sprint
		for offset := 0; ; offset += 200 {
			batch, err := a.store.ListSprints(ctx, key, nil, 200, offset)
			if err != nil {
				return nil, err
			}
			sprints = append(sprints, batch...)
			if len(batch) < 200 {
				break
			}
		}
		found = sprint == "" || sprint == "backlog"
		for _, item := range sprints {
			value := strconv.Itoa(item.Number)
			sprintOptions = append(sprintOptions, Option{Value: value, Label: sprintName(item), Selected: value == sprint})
			if value == sprint {
				found = true
				number := item.Number
				filter.SprintNumber = &number
				view.Subtitle = "Спринт " + sprintName(item)
			}
		}
		if !found {
			return nil, errInvalidFilter
		}
	} else if sprint != "" && sprint != "backlog" {
		return nil, errInvalidFilter
	}
	if sprint == "backlog" {
		filter.Backlog, view.Subtitle = true, "Без спринта"
	}
	view.Selects = append(view.Selects, makeSelect("sprint", "Спринт", sprintOptions))
	view.InSprint = filter.SprintNumber != nil
	selectedStatuses := query["columns"]
	if !query.Has("columns") && !query.Has("columns_set") {
		selectedStatuses = []string{"todo", "in_progress", "review", "done"}
	}
	for _, value := range selectedStatuses {
		if !slices.Contains(statuses, enums.TaskStatus(value)) {
			return nil, errInvalidFilter
		}
	}
	var columnOptions []Option
	for _, status := range statuses {
		selected := slices.Contains(selectedStatuses, string(status))
		columnOptions = append(columnOptions, Option{Value: string(status), Label: statusLabel(status), Selected: selected})
		if selected {
			view.Columns = append(view.Columns, Column{Status: status})
		}
	}
	columnSelect := makeSelect("columns", "Колонки", columnOptions)
	columnSelect.Multiple = true
	columnSelect.Value = fmt.Sprintf("%d из %d", len(view.Columns), len(statuses))
	view.Selects = append(view.Selects, columnSelect)
	kind := query.Get("type")
	if kind != "" {
		value := enums.TaskType(kind)
		if value != enums.TaskTypeFeature && value != enums.TaskTypeBug && value != enums.TaskTypeChore {
			return nil, errInvalidFilter
		}
		filter.Type = &value
	}
	view.Selects = append(view.Selects, makeSelect("type", "Тип задачи", []Option{
		{Value: "", Label: "Все типы", Selected: kind == ""},
		{Value: "feature", Label: "Функция", Selected: kind == "feature"},
		{Value: "bug", Label: "Ошибка", Selected: kind == "bug"},
		{Value: "chore", Label: "Поддержка", Selected: kind == "chore"},
	}))
	var tasks []models.Task
	for offset := 0; ; offset += filter.Limit {
		filter.Offset = offset
		batch, err := a.store.SearchTasks(ctx, filter)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, batch...)
		if len(batch) < filter.Limit {
			break
		}
	}
	slices.SortStableFunc(tasks, func(left, right models.Task) int {
		if delta := priorityRank(left.Priority) - priorityRank(right.Priority); delta != 0 {
			return delta
		}
		return right.UpdatedAt.Compare(left.UpdatedAt)
	})
	taskIDs := make([]string, len(tasks))
	for index, task := range tasks {
		taskIDs[index] = task.ID
	}
	summaries, err := a.store.CodeLinkSummaries(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	codeLinks, err := a.store.ListCodeLinksForTasks(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	counts := make(map[enums.TaskStatus]int)
	var additions, deletions int64
	for _, task := range tasks {
		counts[task.Status]++
		summary := summaries[task.ID]
		view.Stats.CodeChanges.Links += summary.Links
		view.Stats.CodeChanges.WithStats += summary.WithStats
		if summary.Additions != nil {
			additions += *summary.Additions
		}
		if summary.Deletions != nil {
			deletions += *summary.Deletions
		}
		for index := range view.Columns {
			if view.Columns[index].Status == task.Status {
				view.Columns[index].Tasks = append(view.Columns[index].Tasks, Card{Task: task, Assignee: lookupPerson(people, task.AssigneeID), CodeLinks: codeLinks[task.ID], CodeChanges: summary})
				view.Count++
				break
			}
		}
	}
	for _, status := range statuses {
		view.Summary = append(view.Summary, StatusCount{Status: status, Count: counts[status]})
	}
	view.Stats.Done = counts[enums.TaskStatusDone]
	view.Stats.Open = counts[enums.TaskStatusTodo] + counts[enums.TaskStatusInProgress] + counts[enums.TaskStatusReview]
	view.Stats.Cancelled = counts[enums.TaskStatusCancelled]
	view.Stats.Total = view.Stats.Done + view.Stats.Open
	if view.Stats.Total > 0 {
		view.Stats.Completion = (view.Stats.Done*100 + view.Stats.Total/2) / view.Stats.Total
	}
	view.Stats.ChangedLines = additions + deletions
	if view.Stats.CodeChanges.WithStats > 0 {
		view.Stats.CodeChanges.Additions, view.Stats.CodeChanges.Deletions = &additions, &deletions
	}
	page := &Page{Kind: "board", Board: view}
	title := view.Name + " — Agenda"
	path := "/"
	if explicitBoard && boardPattern.MatchString(key) {
		path = "/" + key
		query.Del("board")
	} else if !explicitBoard {
		title = "Agenda — Меньше шума. Больше сделанного."
	}
	page.Meta = a.metadata(r, title, "Agenda — задачи, исполнители и спринты в одном рабочем пространстве. "+view.Subtitle+".", path, query.Encode())
	if path != "/" {
		a.setPreview(&page.Meta, previewCard{Kind: "board", ID: key, Title: view.Name})
	}
	return page, nil
}

func codeLinkLabel(link models.CodeLink) string {
	if link.Type == enums.CodeLinkTypeCommit && link.HeadSHA != nil {
		return (*link.HeadSHA)[:min(7, len(*link.HeadSHA))]
	}
	u, err := url.Parse(link.URL)
	if err == nil {
		parts := strings.Split(strings.TrimRight(u.Path, "/"), "/")
		last := parts[len(parts)-1]
		if link.Type == enums.CodeLinkTypePullRequest {
			if _, err = strconv.ParseUint(last, 10, 64); err == nil {
				return "PR #" + last
			}
		} else if len(last) >= 7 && strings.Trim(last, "0123456789abcdefABCDEF") == "" {
			return last[:7]
		}
	}
	if link.Type == enums.CodeLinkTypePullRequest {
		return "Pull request"
	}
	return "Коммит"
}

func (a *Application) taskPage(r *http.Request, id string) (*Page, error) {
	ctx := r.Context()
	task, err := a.store.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	board, err := a.store.GetBoard(ctx, task.BoardKey, false)
	if err != nil {
		return nil, err
	}
	users, err := a.store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	people := peopleMap(users)
	view := &TaskPage{Task: *task, BoardName: board.Name, Assignee: lookupPerson(people, task.AssigneeID), Creator: lookupPerson(people, &task.CreatedBy), Sprint: "Без спринта"}
	view.CodeLinks, err = a.store.ListCodeLinks(ctx, id)
	if err != nil {
		return nil, err
	}
	summaries, err := a.store.CodeLinkSummaries(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	view.CodeChanges = summaries[id]
	if task.SprintNumber != nil {
		sprint, err := a.store.GetSprint(ctx, task.BoardKey, *task.SprintNumber)
		if err != nil && !errors.Is(err, store.ErrSprintNotFound) {
			return nil, err
		}
		if sprint != nil {
			view.Sprint = sprintName(*sprint)
		}
	}
	if task.ParentID != nil {
		if err = a.addRelated(ctx, view, *task.ParentID, "Родительская"); err != nil {
			return nil, err
		}
	}
	subtasks, err := a.store.ListSubtasks(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, subtask := range subtasks {
		view.Related = append(view.Related, RelatedTask{ID: subtask.ID, Title: subtask.Title, Label: "Подзадача", Status: subtask.Status})
	}
	links, err := a.store.ListTaskLinks(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		other, label := link.ToTaskID, "Связана"
		if link.FromTaskID != id {
			other = link.FromTaskID
		}
		switch link.Type {
		case enums.LinkTypeBlocks:
			label = "Блокирует"
			if link.FromTaskID != id {
				label = "Заблокирована"
			}
		case enums.LinkTypeDuplicates:
			label = "Дублирует"
		}
		if err = a.addRelated(ctx, view, other, label); err != nil {
			return nil, err
		}
	}
	comments, err := a.store.ListComments(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, comment := range comments {
		view.Comments = append(view.Comments, Comment{Comment: comment, Author: lookupPerson(people, &comment.AuthorID)})
	}
	page := &Page{Kind: "task", Task: view}
	description := task.Description
	if strings.TrimSpace(description) == "" {
		description = board.Name + " · " + statusLabel(task.Status) + " · " + typeLabel(task.Type)
	}
	page.Meta = a.metadata(r, task.ID+" · "+task.Title+" — Agenda", description, "/"+task.ID, "")
	a.setPreview(&page.Meta, previewCard{Kind: "task", ID: task.ID, Title: task.Title, Board: board.Name})
	return page, nil
}

func (a *Application) addRelated(ctx context.Context, view *TaskPage, id, label string) error {
	task, err := a.store.GetTask(ctx, id)
	if errors.Is(err, store.ErrTaskNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	view.Related = append(view.Related, RelatedTask{ID: task.ID, Title: task.Title, Label: label, Status: task.Status})
	return nil
}

func makeSelect(name, label string, options []Option) Select {
	result := Select{Name: name, Label: label, Options: options}
	for _, option := range options {
		if option.Selected {
			result.Value = option.Label
			break
		}
	}
	return result
}

func peopleMap(users []models.User) map[string]Person {
	people := make(map[string]Person, len(users))
	for _, user := range users {
		name := personName(user)
		letters := []rune{}
		for _, word := range strings.Fields(name) {
			letters = append(letters, unicode.ToUpper([]rune(word)[0]))
			if len(letters) == 2 {
				break
			}
		}
		if len(letters) == 1 {
			if runes := []rune(name); len(runes) > 1 {
				letters = append(letters, unicode.ToUpper(runes[1]))
			}
		}
		people[user.ID.String()] = Person{Name: name, Initials: string(letters), Username: user.Username}
	}
	return people
}

func personName(user models.User) string {
	if name := strings.TrimSpace(user.Name); name != "" {
		return name
	}
	return user.Username
}

func lookupPerson(people map[string]Person, id *uuid.UUID) Person {
	if id == nil {
		return Person{Name: "Без исполнителя", Initials: "—"}
	}
	if person, ok := people[id.String()]; ok {
		return person
	}
	return Person{Name: "Неизвестный пользователь", Initials: "?"}
}

func sprintName(sprint models.Sprint) string {
	name := fmt.Sprintf("%02d", sprint.Number)
	if sprint.Name != nil && *sprint.Name != "" {
		name += " · " + *sprint.Name
	}
	return name
}

func excerpt(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}

func statusLabel(value enums.TaskStatus) string {
	return map[enums.TaskStatus]string{enums.TaskStatusTodo: "К работе", enums.TaskStatusInProgress: "В работе", enums.TaskStatusReview: "На проверке", enums.TaskStatusDone: "Готово", enums.TaskStatusCancelled: "Отменено"}[value]
}

func typeLabel(value enums.TaskType) string {
	return map[enums.TaskType]string{enums.TaskTypeFeature: "Функция", enums.TaskTypeBug: "Ошибка", enums.TaskTypeChore: "Поддержка"}[value]
}

func priorityLabel(value enums.TaskPriority) string {
	return map[enums.TaskPriority]string{enums.TaskPriorityCritical: "Критический", enums.TaskPriorityHigh: "Высокий", enums.TaskPriorityMedium: "Средний", enums.TaskPriorityLow: "Низкий"}[value]
}

func priorityRank(value enums.TaskPriority) int {
	return map[enums.TaskPriority]int{enums.TaskPriorityCritical: 0, enums.TaskPriorityHigh: 1, enums.TaskPriorityMedium: 2, enums.TaskPriorityLow: 3}[value]
}

func taskCount(count int) string {
	return strconv.Itoa(count) + " " + taskWord(count)
}

func taskWord(count int) string {
	word := "задач"
	if count%100 < 11 || count%100 > 14 {
		if count%10 == 1 {
			word = "задача"
		} else if count%10 >= 2 && count%10 <= 4 {
			word = "задачи"
		}
	}
	return word
}
