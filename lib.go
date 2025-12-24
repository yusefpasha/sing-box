package mobile

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/option"
)

// این اینترفیس باید در کاتلین پیاده‌سازی شود تا Go بتواند اطلاعات را به اندروید بفرستد
type PlatformInterface interface {
	OnStats(uplink int64, downlink int64, memory int64)
	OnLog(message string)
}

var (
	boxInstance *box.Box
	cancelFunc  context.CancelFunc
)

// StartVPN توسط کاتلین صدا زده می‌شود
func StartVPN(configContent string, tunFd int, platform PlatformInterface) string {
	// آزادسازی حافظه قبل از استارت جدید
	debug.FreeOSMemory()

	ctx, cancel := context.WithCancel(context.Background())
	cancelFunc = cancel

	// 1. Parse Config
	var options option.Options
	err := json.Unmarshal([]byte(configContent), &options)
	if err != nil {
		return "Config Error: " + err.Error()
	}

	// 2. Inject Tun File Descriptor
	// نکته مهم: در اندروید بهتر است VpnService فایل دیسکریپتور را بسازد و به Go بدهد.
	// ما باید در کانفیگ sing-box، اینپوت tun را طوری تنظیم کنیم که از این FD استفاده کند.
	// توجه: این بخش نیازمند دستکاری تنظیمات Inbound در سطح کد است.
    // برای سادگی اینجا فرض میکنیم کانفیگ شما صحیح است، اما در عمل باید options.Inbounds را اصلاح کنید.

	// 3. Create Box
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		return "Create Box Error: " + err.Error()
	}
	boxInstance = instance

	// 4. Start
	err = instance.Start()
	if err != nil {
		return "Start Error: " + err.Error()
	}

	// 5. Start Stats Loop (Goroutine)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// دریافت آمار از هسته sing-box
                // توجه: متدهای دریافت آمار در نسخه‌های مختلف sing-box متفاوت است.
                // فرض بر استفاده از سرویس Stats هسته است.
                // این یک مثال شماتیک است:
				// up, down := instance.Router().GetTraffic()
                // mem := getMemoryUsage()
				platform.OnStats(0, 0, 0) // مقادیر واقعی را جایگزین کنید
			}
		}
	}()

	return "" // Empty string means success
}

func StopVPN() {
	if cancelFunc != nil {
		cancelFunc()
	}
	if boxInstance != nil {
		boxInstance.Close()
		boxInstance = nil
	}
}