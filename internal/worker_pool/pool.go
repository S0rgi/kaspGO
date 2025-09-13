package worker_pool

import "sync"

// Команда разработки решений для управления корпоративной защитой

// Выполненное задание направлять по ссылке - https://box.kaspersky.com/u/d/dffa4ef2a42c4e7caecc/

// Важно! Не забудьте указать ваши фамилию и имя в названии файла.

// Необходимо реализовать WorkerPool и покрыть тестами.

type WorkerPool struct {
	tasks    chan func()    // очередь задач
	wg       sync.WaitGroup // ждём завершения задач
	stopOnce sync.Once      // защита от двойного вызова Stop
	stopCh   chan struct{}  // сигнал воркерам остановиться
	closed   bool           // признак закрытия пула
	mu       sync.Mutex     // защита от гонок при закрытии и добавлении задач
}

func NewWorkerPool(numWorkers int) Pool {
	wp := &WorkerPool{
		tasks:  make(chan func()),
		stopCh: make(chan struct{}),
	}

	// запускаем воркеров
	for i := 0; i < numWorkers; i++ {
		go wp.worker()
	}

	return wp
}
func (wp *WorkerPool) worker() {
	for {
		select {
		case <-wp.stopCh:
			return
		case task := <-wp.tasks:
			if task == nil {
				return
			}
			task()
			wp.wg.Done()
		}
	}
}

// Submit - добавить таску в воркер пул
func (wp *WorkerPool) Submit(task func()) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	if wp.closed {
		return // или panic("submit to closed pool")
	}
	wp.wg.Add(1)
	wp.tasks <- task
}

// SubmitWait - добавить таску в воркер пул и дождаться окончания ее выполнения
func (wp *WorkerPool) SubmitWait(task func()) {
	var inner sync.WaitGroup
	inner.Add(1)

	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return // или panic("submit to closed pool")
	}
	wp.wg.Add(1)
	wp.mu.Unlock()

	wp.tasks <- func() {
		defer inner.Done()
		task()
	}

	inner.Wait()
}

// Stop - остановить воркер пул, дождаться выполнения только тех тасок, которые выполняются сейчас
func (wp *WorkerPool) Stop() {
	wp.stopOnce.Do(func() {
		wp.mu.Lock()
		wp.closed = true
		close(wp.stopCh)
		wp.mu.Unlock()
	})
}

// StopWait - остановить воркер пул, дождаться выполнения всех тасок, даже тех, что не начали выполняться, но лежат в очереди
func (wp *WorkerPool) StopWait() {
	wp.Stop()
	wp.wg.Wait()
}
