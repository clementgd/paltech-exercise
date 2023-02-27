package telegram_bot

import (
	"errors"
	"log"
	"strconv"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	. "paltech.robot/robot"
)

type TelegramBot struct {
	apiBot *tgbotapi.BotAPI
	recipientsChatIdSet map[int64]bool
	recipientsMutex sync.Mutex
	Robots *[]*Robot
	RobotsMutex *sync.Mutex
}

func NewTelegramBot(apiKey string) (bot *TelegramBot, err error) {
	bot = new(TelegramBot)
	bot.apiBot, err = tgbotapi.NewBotAPI(apiKey)
	log.Printf("Authorized on account %s", *(&bot.apiBot.Self.UserName))
	bot.recipientsChatIdSet = make(map[int64]bool)
	return bot, err
}

func (bot *TelegramBot) getUpdateMessageAndImagePathForRobot(id int) (string, string, error) {
	bot.RobotsMutex.Lock()
	if id >= len(*bot.Robots) {
		bot.RobotsMutex.Unlock()
		return "", "", errors.New("Hum, I can't find bot " + strconv.Itoa(id) + ", are you sure it was created ?")
	}

	robot := (*bot.Robots)[id]
	robotId := strconv.Itoa(robot.Id)
	statusHistoryLength := strconv.Itoa(len(robot.StatusHistory))
	latestStatus := (*bot.Robots)[id].GetLatestStatus()
	bot.RobotsMutex.Unlock()

	message := "Status of robot " + robotId + " :\n"
	message += " - Completion : " + strconv.Itoa(latestStatus.WaypointsReached) + "/" 
	message += strconv.Itoa(latestStatus.WaypointsTotal) + " waypoints reached\n"
	message += " - Distance covered : " + strconv.FormatFloat(latestStatus.DistanceCovered, 'f', 1, 64) + "m\n"

	filepath := "pathImages/path-" + robotId + "-" + statusHistoryLength + ".png"

	return message, filepath, nil
}

func (bot *TelegramBot) sendText(chatId int64, text string) {
	textMessage := tgbotapi.NewMessage(chatId, text)
	if _, err := bot.apiBot.Send(textMessage); err != nil {
		log.Panic(err)
	}
}

func (bot *TelegramBot) sendImage(chatId int64, imagePath string) {
	imageMessage := tgbotapi.NewPhoto(chatId, tgbotapi.FilePath(imagePath))
	if _, err := bot.apiBot.Send(imageMessage); err != nil {
		log.Panic(err)
	}
}

func (bot *TelegramBot) respondHelp(incomingMessage *tgbotapi.Message) {
	bot.sendText(
		incomingMessage.Chat.ID,
		"Send /status-<bot id> to get an immediate status update on bot with id <bot id>",
	)
}

func (bot *TelegramBot) respondStatusUpdate(incomingMessage *tgbotapi.Message) {
	args := incomingMessage.CommandArguments()
	chatId := incomingMessage.Chat.ID
	robotId, err := strconv.Atoi(args)
	if err != nil {
		bot.sendText(chatId, "I'm having trouble reading " + args + " as an integer")
		return
	}

	message, imagepath, err := bot.getUpdateMessageAndImagePathForRobot(robotId)
	if err != nil {
		bot.sendText(chatId, err.Error())
		return
	}

	bot.sendText(chatId, message)
	bot.sendImage(chatId, imagepath)
}


func (bot *TelegramBot) registerRecipient(incomingMessage *tgbotapi.Message) {
	bot.recipientsMutex.Lock()
	bot.recipientsChatIdSet[incomingMessage.Chat.ID] = true
	bot.recipientsMutex.Unlock()
	bot.sendText(
		incomingMessage.Chat.ID, 
		"You registered successfully. You will now receive periodic updates.\n" +
		"You can send /status-<bot id> to get an immediate status update on bot <bot id>",
	)
}

func (bot *TelegramBot) unregisterRecipient(incomingMessage *tgbotapi.Message) {
	bot.recipientsMutex.Lock()
	delete(bot.recipientsChatIdSet, incomingMessage.Chat.ID)
	bot.recipientsMutex.Unlock()
	bot.sendText(incomingMessage.Chat.ID, "You unregistered successfully. You will not receive any more updates.")
}

func (bot *TelegramBot) processUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

    updates := bot.apiBot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message == nil {
            continue
        }

        if !update.Message.IsCommand() {
            continue
        }

        switch update.Message.Command() {
        case "status":
			go bot.respondStatusUpdate(update.Message)
		case "start":
			go bot.registerRecipient(update.Message)
		case "stop":

        default:
            go bot.respondHelp(update.Message)
        }
    }
}

func (bot *TelegramBot) ListenAndServe() {
	go bot.processUpdates()
}



func (bot *TelegramBot) SendPeriodicUpdateForRobot(robotId int, isRobotBackOnlineMessage bool) {
	message, imagepath, err := bot.getUpdateMessageAndImagePathForRobot(robotId)
	if err != nil {
		log.Panic(err)
		return
	}

	if isRobotBackOnlineMessage {
		message = "Robot " + strconv.Itoa(robotId) + " came back online !\n" + message
	}

	bot.recipientsMutex.Lock()
	for recipientChatId := range bot.recipientsChatIdSet {
		bot.sendText(recipientChatId, message)
		bot.sendImage(recipientChatId, imagepath)
	}
	bot.recipientsMutex.Unlock()
}

func (bot *TelegramBot) SendTimeoutMessage(robotId int) {
	message := "Robot " + strconv.Itoa(robotId) + " timed out !"

	// TODO Maybe move to a broadcastRecipients function ?
	bot.recipientsMutex.Lock()
	for recipientChatId := range bot.recipientsChatIdSet {
		bot.sendText(recipientChatId, message)
	}
	bot.recipientsMutex.Unlock()
}