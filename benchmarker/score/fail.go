package score

import (
	"sort"
	"sync"
)

type failErrors struct {
	sync.RWMutex
	errs []error
}

var failInstance *failErrors
var failOnce sync.Once

func GetFailErrorsInstance() *failErrors {
	failOnce.Do(func() {
		errs := make([]error, 0)
		failInstance = &failErrors{errs: errs}
	})

	return failInstance
}

func GetFailErrors() []error {
	sort.Sort(GetFailErrorsInstance())
	var tmp string
	retErrs := make([]error, 0)

	// 適当にuniqする
	for _, e := range failInstance.errs {
		if tmp != e.Error() {
			tmp = e.Error()
			retErrs = append(retErrs, e)
		}
	}
	return retErrs
}

func (fes failErrors) Len() int {
	return len(fes.errs)
}

func (fes failErrors) Swap(i, j int) {
	fes.errs[i], fes.errs[j] = fes.errs[j], fes.errs[i]
}

func (fes failErrors) Less(i, j int) bool {
	return fes.errs[i].Error() < fes.errs[j].Error()
}

func (fes *failErrors) Append(e error) {
	fes.Lock()
	fes.errs = append(fes.errs, e)
	fes.Unlock()
}
