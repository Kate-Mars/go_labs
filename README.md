# TripGo — репозиторий для лабораторных работ

Заготовка курса «Разработка микросервисов на Go». Здесь вы делаете все пять
работ: каждая следующая продолжает предыдущую, переписывать сервис с нуля не
нужно.

## Что делать сразу

1. **Fork** этого репозитория к себе.
2. Переименуйте форк в `trip-service`: Settings → Repository name. Так адрес
   репозитория совпадёт с путём модуля, и импорты не придётся править потом.
3. Заведите модуль и прочитайте, что нужно до первой работы:

```bash
git clone git@github.com:<ваш-логин>/trip-service.git
cd trip-service
go mod init github.com/<ваш-логин>/trip-service
```

Дальше — [`docs/getting-started.md`](https://github.com/course-go-autumn-2026/course/blob/main/docs/getting-started.md)
в репозитории курса: инструменты, окружение, миграции, вид сданной работы.

## Где что лежит

| Что | Где |
|---|---|
| Задания, документация, контракты | [`course-go-autumn-2026/course`](https://github.com/course-go-autumn-2026/course) |
| Слайды и записи лекций | [`course/lections`](https://github.com/course-go-autumn-2026/course/tree/main/lections) |
| Как оценивают, дедлайны, порядок сдачи | [`course/docs/grading.md`](https://github.com/course-go-autumn-2026/course/blob/main/docs/grading.md) |
| Локальное окружение и утилита `tripgoctl` | [`course-go-autumn-2026/course-infra`](https://github.com/course-go-autumn-2026/course-infra) |

Задания появляются по мере курса, каждое — после своей пары лекций.

## Как сдавать

Ветка `homework/NN` от `main`, pull request в `main` своего форка, ссылка
ментору до дедлайна. Подробно — в `grading.md` репозитория курса.

## Чужие работы

Форки видны всем, включая ваши. Смотреть чужие решения, пока идёт курс, —
плохая идея: одинаковый код виден сразу, а разбираться на защите придётся
самому.
