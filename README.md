# kaspGO

### Выполненное задание: WorkerPool

Реализация задания и тесты находятся здесь:

- Код и тесты: `internal/worker_pool/`
- Описание реализации и инструкции: `internal/worker_pool/README.md`

Кратко о задаче:

- Реализован классический пул воркеров с методами `Submit`, `SubmitWait`, `Stop`, `StopWait`.
- Покрыт тестами, включая сценарии остановки и ожидания.

### Запуск тестов и покрытие

Через Makefile:

```bash
make coverage        # тесты + покрытие (вывод в консоль)
make coverage-html   # HTML-отчёт (coverage/coverage.html)
```

### Документация

Сгенерировать сводную документацию по проекту:

```bash
make doc
```

Результат: `docs/DOCUMENTATION.md`.

### CI

Тесты запускаются автоматически при пуше/PR:

- Workflow: `.github/workflows/tests.yml`