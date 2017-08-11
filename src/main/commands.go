package main

import (
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	r "gopkg.in/gorethink/gorethink.v3"
	"gopkg.in/resty.v0"
	"github.com/Dimonchik0036/vk-api"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func StartCommand(update vkapi.LPUpdate) {
	msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Напишите команду /help, чтобы узнать всевозможные команды.")
	client.SendMessage(msg)
}

func HelpCommand(update vkapi.LPUpdate) {
	msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Список команд:\n/today - расписание на сегодня.\n/tomorrow - расписание на завтра.\n/get 0-6 - расписание на нужный день.\nНапример /get 3 - на среду.\n\nВсе эти данные можно также получить посредством inline-режима!\n\n/save - сохраняет вашу группу и её расписание.\n/update - обновляет расписание вашей группы.\n/delete - полностью удаляет ваш профиль из бота.\n/status - отображает текущий статус.")
	client.SendMessage(msg)
}

func Update(groupNum string, userId int64) error {
	// Делаем запрос, чтобы получить внутренний ID группы на основе её номера
	resp, err := resty.R().SetQueryParams(map[string]string{
		"p_p_id":          "pubStudentSchedule_WAR_publicStudentSchedule10",
		"p_p_lifecycle":   "2",
		"p_p_resource_id": "getGroupsURL",
		"query":           groupNum,
	}).Get("https://kai.ru/raspisanie")
	if err != nil {
		return err
	}

	// Достаем ID группы, из полученного JSON
	groupId := gjson.Get(resp.String(), "0.id").Num

	// Делаем запрос, чтобы получить расписание группы, на основе полученного ID
	resp, err = resty.R().SetQueryParams(map[string]string{
		"p_p_id":          "pubStudentSchedule_WAR_publicStudentSchedule10",
		"p_p_lifecycle":   "2",
		"p_p_resource_id": "schedule",
	}).SetFormData(map[string]string{
		"groupId": fmt.Sprint(groupId),
	}).Post("https://kai.ru/raspisanie")
	if err != nil {
		return err
	}

	schedule := resp.String()

	if len(schedule) > 2 {
		// Добавляем в базу пустую запись о новой группе
		_, err = r.Table("groups").Insert(map[string]interface{}{
			"id":       groupId,
			"schedule": make([]interface{}, 0),
			"time":     r.Now(),
		}, r.InsertOpts{
			Conflict: "update",
		}).RunWrite(session)
		if err != nil {
			log.Println(err)
		}

		// Цикл по дням недели
		for i := 1; i <= 6; i++ {
			dayNum := fmt.Sprint(i) + "."

			// Создаем массив для хранения занятий за день
			dayArray := make([]map[string]string, 0)

			// Цикл по занятиям
			subjectsCount := gjson.Get(schedule, dayNum+"#")
			for j := 0; j < int(subjectsCount.Int()); j++ {
				subjectNum := fmt.Sprint(j) + "."

				// Достаем все нужные поля из JSON, а затем удаляем все лишние символы
				subjectTime := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"dayTime").Str)
				subjectWeek := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"dayDate").Str)
				subjectName := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"disciplName").Str)
				subjectType := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"disciplType").Str)
				buildNum := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"buildNum").Str)
				cabinetNum := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"audNum").Str)
				teacherName := strings.TrimSpace(gjson.Get(schedule, dayNum+subjectNum+"prepodName").Str)

				// Добавляем к существующему массиву новое занятие
				dayArray = append(dayArray, map[string]string{
					"subjectTime": subjectTime,
					"subjectWeek": subjectWeek,
					"subjectName": subjectName,
					"subjectType": subjectType,
					"buildNum":    buildNum,
					"cabinetNum":  cabinetNum,
					"teacherName": teacherName,
				})
			}

			// Добавляем в базу день
			_, err := r.Table("groups").Get(groupId).Update(map[string]interface{}{
				"schedule": r.Row.Field("schedule").InsertAt(i-1, dayArray),
			}).RunWrite(session)
			if err != nil {
				log.Println(err)
			}
		}

		// Добавляем в базу запись о пользователе
		_, err = r.Table("users").Insert(map[string]interface{}{
			"id":       userId,
			"groupId":  groupId,
			"groupNum": groupNum,
		}, r.InsertOpts{
			Conflict: "update",
		}).RunWrite(session)
		if err != nil {
			log.Println(err)
		}

		return nil
	} else {
		return errors.New("Too short schedule!")
	}
}

func SaveCommand(update vkapi.LPUpdate) {
	re := regexp.MustCompile(`\s(.+)`)

	groupNum := re.FindStringSubmatch(update.Message.Text)
	if len(groupNum) > 0 {
		err := Update(groupNum[1], update.Message.FromID)
		if err != nil {
			log.Println(err)
		}

		if err == nil {
			var msg vkapi.MessageConfig
			if err == nil {
				msg = vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Cохранено!")
			} else {
				msg = vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Что-то пошло не так...")
			}
			client.SendMessage(msg)
		} else {
			msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Похоже введен неверный номер группы.")
			client.SendMessage(msg)
		}
	} else {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Пример: <code>/save 4108</code>, чтобы сохранить группу 4108. Замените этот номер на любой другой.")
		client.SendMessage(msg)
	}
}

func GetDayName(day int) string {
	switch day {
	case 0:
		return "Понедельник"
	case 1:
		return "Вторник"
	case 2:
		return "Среда"
	case 3:
		return "Четверг"
	case 4:
		return "Пятница"
	case 5:
		return "Суббота"
	case 6:
		return "Воскресенье"
	default:
		return "ОШИБКА!"
	}
}

func GetDayText(subjects []map[string]string) string {
	text := ""

	// Цикл по занятиям
	for _, elem := range subjects {
		// Добавляем к существующему сообщению новое занятие
		text += fmt.Sprintf("%s %s\n%s\n%s, %s, %s, %s\n\n",
			elem["subjectTime"],
			elem["subjectWeek"],
			elem["subjectName"],
			elem["subjectType"],
			elem["buildNum"],
			elem["cabinetNum"],
			elem["teacherName"],
		)
	}

	return text
}

func FullCommand(update vkapi.LPUpdate) {
	// Получаем из базы нужную информацию
	user, err := GetUser(update.Message.FromID)
	if err != nil {
		log.Println(err)
	}

	group, err := GetGroup(user.GroupID)
	if err != nil {
		log.Println(err)
	}

	if err == nil {
		// Инициализируем пустое сообщение
		text := ""

		// Цикл по дням недели
		for i, _ := range group.Schedule {
			// Добавляем к существующему сообщению день недели
			text += GetDayName(i) + "\n"
			text += GetDayText(group.Schedule[i])
		}

		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), text)
		client.SendMessage(msg)
	} else {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Что-то пошло не так...")
		client.SendMessage(msg)
	}
}

func TodayCommand(update vkapi.LPUpdate) {
	// Получаем номер текущего дня
	day := int(time.Now().Weekday()) - 1

	if day != 6 {
		// Получаем из базы нужную информацию
		user, err := GetUser(update.Message.FromID)
		if err != nil {
			log.Println(err)
		}

		group, err := GetGroup(user.GroupID)
		if err != nil {
			log.Println(err)
		}

		if err == nil {
			// Инициализируем пустое сообщение
			text := ""

			text += GetDayName(day) + "\n"
			text += GetDayText(group.Schedule[day])

			msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), text)
			client.SendMessage(msg)
		} else {
			msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Что-то пошло не так...")
			client.SendMessage(msg)
		}
	} else if day == 6 {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Похоже это воскресенье.")
		client.SendMessage(msg)
	}
}

func TomorrowCommand(update vkapi.LPUpdate) {
	// Получаем номер завтрашнего дня
	day := int(time.Now().Weekday())

	if day != 6 {
		// Получаем из базы нужную информацию
		user, err := GetUser(update.Message.FromID)
		if err != nil {
			log.Println(err)
		}

		group, err := GetGroup(user.GroupID)
		if err != nil {
			log.Println(err)
		}

		if err == nil {
			// Инициализируем пустое сообщение
			text := ""

			text += GetDayName(day) + "\n"
			text += GetDayText(group.Schedule[day])

			msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), text)
			client.SendMessage(msg)
		} else {
			msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Что-то пошло не так...")
			client.SendMessage(msg)
		}
	} else {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Похоже это воскресенье.")
		client.SendMessage(msg)
	}
}

func GetCommand(update vkapi.LPUpdate) {
	re := regexp.MustCompile(`\s(.+)`)
	dayStr := re.FindStringSubmatch(update.Message.Text)
	if len(dayStr) > 0 {
		day, err := strconv.ParseInt(dayStr[1], 10, 32)
		if err != nil {
			log.Println(err)
		}

		day--

		if err == nil && day > -1 && day < 6 {
			// Получаем из базы нужную информацию
			user, err := GetUser(update.Message.FromID)
			if err != nil {
				log.Println(err)
			}

			group, err := GetGroup(user.GroupID)
			if err != nil {
				log.Println(err)
			}

			if err == nil {
				// Инициализируем пустое сообщение
				text := ""

				text += GetDayName(int(day)) + "\n"
				text += GetDayText(group.Schedule[day])

				msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), text)
				client.SendMessage(msg)
			} else {
				msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Что-то пошло не так...")
				client.SendMessage(msg)
			}
		} else if day == 6 {
			msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Похоже это воскресенье.")
			client.SendMessage(msg)
		} else {
			msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Укажите правильный порядковый номер дня недели!")
			client.SendMessage(msg)
		}
	} else {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Пример: <code>/get 3</code>, чтобы получить расписание на среду.")
		client.SendMessage(msg)
	}
}

func StatusCommand(update vkapi.LPUpdate) {
	// Получаем из базы нужную информацию
	user, err := GetUser(update.Message.FromID)
	if err != nil {
		log.Println(err)
	}

	group, err := GetGroup(user.GroupID)
	if err != nil {
		log.Println(err)
	}

	if err == nil {
		// Инициализируем пустое сообщение
		text := ""

		text += "ID: " + fmt.Sprint(user.Id) + "\n"
		text += "Группа: " + user.GroupNum + "\n"
		text += "Последнее обновление: " + fmt.Sprint(group.Time)

		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), text)
		client.SendMessage(msg)
	} else {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Что-то пошло не так...")
		client.SendMessage(msg)
	}
}

func UpdateCommand(update vkapi.LPUpdate) {
	// Получаем из базы нужную информацию
	user, err := GetUser(update.Message.FromID)
	if err != nil {
		log.Println(err)
	}

	if err == nil {
		err = Update(fmt.Sprint(user.GroupNum), user.Id)
		if err != nil {
			log.Println(err)
		}

		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Обновлено!")
		client.SendMessage(msg)
	} else {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Что-то пошло не так...")
		client.SendMessage(msg)
	}
}

func DeleteCommand(update vkapi.LPUpdate) {
	_, err := r.Table("users").Get(update.Message.FromID).Delete().RunWrite(session)
	if err != nil {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Что-то пошло не так...")
		client.SendMessage(msg)
	} else {
		msg := vkapi.NewMessage(vkapi.NewDstFromUserID(update.Message.FromID), "Удалено!")
		client.SendMessage(msg)
	}
}
