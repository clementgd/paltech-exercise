module paltech.server

go 1.20

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/labstack/echo/v4 v4.10.2
	paltech.robot/robot v0.0.0-00010101000000-000000000000
	paltech.telegram_bot/telegram_bot v0.0.0-00010101000000-000000000000
)

require (
	github.com/labstack/gommon v0.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
)

replace paltech.telegram_bot/telegram_bot => ../telegram_bot

replace paltech.robot/robot => ../robot
