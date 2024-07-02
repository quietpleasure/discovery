package retryer

import (
	"context"
	"fmt"
	"time"

	"github.com/quietpleasure/discovery"
	"github.com/quietpleasure/discovery/consul"
)

type FuncExecutor func(ctx context.Context, instanceID string, cfgService *consul.ServiceConfig, cfgConsul *consul.ConsulConfig) (*consul.Registry, error)

type Feedback struct {
	Error   error
	Message string
}

func retry(function FuncExecutor, feedback chan Feedback, maxAttempts ...int) FuncExecutor {
	var max int
	if len(maxAttempts) > 0 {
		max = maxAttempts[0]
	}
	return func(ctx context.Context, instanceID string, cfgService *consul.ServiceConfig, cfgConsul *consul.ConsulConfig) (*consul.Registry, error) {
		attempt := 1
		for {
			reg, err := function(ctx, instanceID, cfgService, cfgConsul)
			if err == nil {
				feedback <- Feedback{
					Message: fmt.Sprintf("retry attempt %d successful", attempt),
				}
				return reg, nil
			}
			if attempt == max && max != 0 {
				feedback <- Feedback{
					Message: "all attempts used",
				}
				return reg, err
			}
			delay := time.Second << uint(attempt)
			feedback <- Feedback{
				Error:   err,
				Message: fmt.Sprintf("retry attempt %d failed repeat after %s", attempt, delay),
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			attempt++
		}
	}
}

func CheckHealthAndReconnect(ctx context.Context, instanceID string, reg discovery.Registry, serviceCfg *consul.ServiceConfig, logFeedback chan Feedback, maxAttempts ...int) {
	feedback := make(chan Feedback)
	defer close(feedback)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := reg.ReportHealthyState("", instanceID); err != nil {
				//отвалился коннект к Консулу, нужно переподключать
				// log.Printf("trying new make registry and register | ERORR: %s\n", err)
				logFeedback <- Feedback{
					Error:   err,
					Message: "trying new make registry and register",
				}
				var max int
				if maxAttempts != nil {
					max = maxAttempts[0]
				}
				retryFunc := retry(consul.MakeRegistryAndRegisterService, feedback, max)
				var (
					newreg *consul.Registry
					rerr   error
				)
				go func() {
					newreg, rerr = retryFunc(ctx, instanceID, serviceCfg, nil)
				}()
				for f := range feedback {
					// if f.Error != nil {
					// 	log.Printf("%s | ERROR: %s\n", f.Message, f.Error)						
					// } else {
					// 	log.Println(f.Message)
					// }
					logFeedback <- f
				}
				// "successful new consul connect" or all attempts used with error
				if rerr == nil {
					reg = newreg
				} else {
					//вышли все попытки подключения нет смыслы в работе сервиса
					panic(err)
				}

			}
		}
		time.Sleep(time.Second)
	}
}
