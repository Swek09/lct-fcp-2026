package assessment

import (
	"fmt"
	"math"
	"math/rand/v2"
)

// Every numeric family states its assumptions and rounding rule explicitly.
// family 1 and 2 are different operations, not just shuffled answer options.
func expandedScenario(spec, grade string, slot, family int, r *rand.Rand, topic string) PrivateQuestion {
	n, m := r.IntN(12)+4, r.IntN(6)+2
	prompt, answer := "", ""
	if slot == 1 {
		switch spec + "-" + grade {
		case "backend-junior":
			if family == 1 {
				prompt = fmt.Sprintf("В orders %d строки. У %d строк customer_id = NULL, у остальных значение заполнено. Что вернёт SELECT COUNT(customer_id) FROM orders?", n+m, m)
				answer = fmt.Sprint(n)
			} else {
				prompt = fmt.Sprintf("В orders %d оплаченных заказов по %d00 ₽ и 3 отменённых заказа. Что вернёт SELECT SUM(total) FROM orders WHERE status = 'paid'?", n, m)
				answer = fmt.Sprint(n * m * 100)
			}
		case "backend-middle":
			if family == 1 {
				prompt = fmt.Sprintf("Есть %d клиентов, у каждого %d заказа. После INNER JOIN orders выполняется COUNT(DISTINCT customers.id). Других клиентов нет. Каков результат?", n, m)
				answer = fmt.Sprint(n)
			} else {
				prompt = fmt.Sprintf("Для пагинации OFFSET равен %d, LIMIT равен %d. В отфильтрованном и отсортированном наборе %d строк. Сколько строк вернёт запрос?", n, m, n+m+3)
				answer = fmt.Sprint(m)
			}
		case "backend-senior":
			if family == 1 {
				prompt = fmt.Sprintf("На складе %d единиц. %d конкурентных транзакций выполняют UPDATE stock SET quantity=quantity-1 WHERE quantity>0. Каждая выполняет один UPDATE, изменения фиксируются, ошибок сериализации нет. Сколько транзакций обновят строку?", n, n+m)
				answer = fmt.Sprint(n)
			} else {
				prompt = fmt.Sprintf("В таблице %d различных ключей. Пакет содержит %d новых ключей по 3 копии каждого и по 2 копии всех старых. ON CONFLICT DO NOTHING, уникальность ключа обеспечена. Сколько новых строк добавится?", n, m)
				answer = fmt.Sprint(m)
			}
		case "frontend-junior":
			if family == 1 {
				prompt = fmt.Sprintf("Родитель имеет область содержимого %d00 px. Дочерний блок width:50%%, box-sizing:border-box, без margin. Какова внешняя ширина дочернего блока в px?", n)
				answer = fmt.Sprint(n * 50)
			} else {
				prompt = fmt.Sprintf("В строке flex %d блока шириной по 40 px, gap равен %d px. Ни padding, ни margin, ни border нет. Какова суммарная ширина блоков с промежутками?", n, m)
				answer = fmt.Sprint(n*40 + (n-1)*m)
			}
		case "frontend-middle":
			if family == 1 {
				prompt = fmt.Sprintf("Контейнер имеет ширину %d px, border-box, padding по 20 px с двух сторон, без border. Grid содержит %d одинаковых колонок и gap 12 px. Какова ширина колонки? Округлите до двух знаков.", n*100, m)
				answer = fmt.Sprintf("%.2f", float64(n*100-40-(m-1)*12)/float64(m))
			} else {
				prompt = fmt.Sprintf("В grid-template-columns: repeat(auto-fill,minmax(100px,1fr)) задан gap 10 px. Область содержимого шириной %d px. Сколько колонок поместится?", n*100)
				answer = fmt.Sprint((n*100 + 10) / 110)
			}
		case "frontend-senior":
			if family == 1 {
				prompt = fmt.Sprintf("Карточки должны быть не уже 180 px, gap 16 px. Контейнер имеет область содержимого %d px. Каково максимальное число колонок одинаковых карточек?", n*100)
				answer = fmt.Sprint((n*100 + 16) / 196)
			} else {
				prompt = fmt.Sprintf("Изображение имеет aspect-ratio:16/9 и ширину %d px. Ограничений высоты и border нет. Какова его высота в px?", n*160)
				answer = fmt.Sprint(n * 90)
			}
		case "analytics-junior":
			if family == 1 {
				prompt = fmt.Sprintf("У продукта %d пользователей, выручка за тот же период %d ₽. Каков ARPU в рублях? Все пользователи входят в знаменатель.", n*100, n*m*100)
				answer = fmt.Sprint(m)
			} else {
				prompt = fmt.Sprintf("%d пользователей сделали по %d заказа каждый. Покупатель определяется как уникальный пользователь с заказом. Сколько покупателей?", n, m)
				answer = fmt.Sprint(n)
			}
		case "analytics-middle":
			if family == 1 {
				prompt = fmt.Sprintf("Конверсия изменилась с %d%% до %d%%. На сколько процентных пунктов она выросла? Введите число без знака процента.", n, n+m)
				answer = fmt.Sprint(m)
			} else {
				prompt = fmt.Sprintf("В когорте A %d пользователей с конверсией 10%%, в B %d пользователей с конверсией 20%%. Какова общая конверсия? Округлите до двух знаков.", n*100, m*100)
				answer = fmt.Sprintf("%.2f", float64(n*10+m*20)/float64(n+m))
			}
		case "analytics-senior":
			if family == 1 {
				prompt = fmt.Sprintf("Для стандартизации эффекта веса сегментов A и B зафиксированы как 0,25 и 0,75. Эффекты в сегментах — %d и %d процентных пунктов. Каков взвешенный эффект? Округлите до двух знаков.", n, m)
				answer = fmt.Sprintf("%.2f", 0.25*float64(n)+0.75*float64(m))
			} else {
				prompt = fmt.Sprintf("В A доля события 10%% при %d наблюдениях, в B 20%% при %d наблюдениях. Если каждое наблюдение имеет одинаковый вес, какова общая доля события в процентах? Округлите до двух знаков.", n*100, m*100)
				answer = fmt.Sprintf("%.2f", float64(n*10+m*20)/float64(n+m))
			}
		}
	} else if slot == 3 {
		switch spec + "-" + grade {
		case "backend-junior":
			if family == 1 {
				prompt = fmt.Sprintf("У пользователя %d документов, у другого %d документов. Endpoint фильтрует owner_id по текущему пользователю. Сколько документов получит первый пользователь?", n, m)
				answer = fmt.Sprint(n)
			} else {
				prompt = fmt.Sprintf("Из %d токенов подтверждения %d уже использованы. Неиспользованные токены действуют, каждый может сработать ровно один раз. Сколько успешных подтверждений ещё возможно?", n+m, m)
				answer = fmt.Sprint(n)
			}
		case "backend-middle":
			if family == 1 {
				prompt = fmt.Sprintf("У компании %d откликов с согласием на контакт. Из них %d отозваны; других оснований доступа нет. Отозванные входят в исходное число. Сколько email можно раскрыть сейчас?", n+m, m)
				answer = fmt.Sprint(n)
			} else {
				prompt = fmt.Sprintf("API проверяет членство в организации и право на объект. Пользователь имеет оба права на %d объекта, только членство на %d других. Сколько объектов он вправе прочитать?", n, m)
				answer = fmt.Sprint(n)
			}
		case "backend-senior":
			if family == 1 {
				prompt = fmt.Sprintf("Ключ подписи A отозван. Есть %d действующих по сроку токенов ключа A и %d токенов ключа B. Верификатор доверяет только B. Сколько токенов пройдут проверку подписи?", n, m)
				answer = fmt.Sprint(m)
			} else {
				prompt = fmt.Sprintf("У пользователя %d сессий версии 8. В БД auth_version=9. После смены версии выдано %d сессий версии 9, все существуют и действуют. Сколько сессий авторизуют пользователя при обязательном совпадении версии?", n, m)
				answer = fmt.Sprint(m)
			}
		case "frontend-junior":
			if family == 1 {
				prompt = fmt.Sprintf("В форме %d кнопок: %d имеют disabled, остальные обычные активные button без отрицательного tabindex. Сколько кнопок входят в Tab-порядок?", n+m, m)
				answer = fmt.Sprint(n)
			} else {
				prompt = fmt.Sprintf("Есть %d полей input. У %d полей задан корректный aria-label, у остальных нет label, aria-labelledby и других средств задания имени. Сколько полей имеют accessible name?", n+m, n)
				answer = fmt.Sprint(n)
			}
		case "frontend-middle":
			if family == 1 {
				prompt = fmt.Sprintf("Диалог содержит %d активных элементов в циклическом Tab-порядке. Фокус на первом. После %d нажатий Tab на каком порядковом элементе окажется фокус? Нумерация с 1, после последнего элемента фокус возвращается к первому.", n, m)
				answer = fmt.Sprint(m%n + 1)
			} else {
				prompt = fmt.Sprintf("%d полей формы невалидны. Из них %d имеют текст ошибки, связанный через aria-describedby; остальных связей нет. Сколько невалидных полей пока не имеют такой связи?", n+m, m)
				answer = fmt.Sprint(n)
			}
		case "frontend-senior":
			if family == 1 {
				prompt = fmt.Sprintf("В виртуальной таблице всего %d строк данных, одновременно в DOM %d. Заголовок имеет отдельную строку. Какое aria-rowcount нужно указать для общего числа строк, включая заголовок?", n*100, m)
				answer = fmt.Sprint(n*100 + 1)
			} else {
				prompt = fmt.Sprintf("В виртуальной таблице заголовок имеет aria-rowindex=1. Окно начинается со строки данных с порядковым номером %d (счёт данных с 1) и содержит %d последовательных строк данных. Какой aria-rowindex у последней видимой строки данных? Заголовок учитывается в общей нумерации.", n*10, m)
				answer = fmt.Sprint(n*10 + m)
			}
		case "analytics-junior":
			if family == 1 {
				prompt = fmt.Sprintf("В отчёте %d событий. %d события имеют некорректную дату и исключены. Остальные валидны. Сколько событий входит в отчёт?", n+m, m)
				answer = fmt.Sprint(n)
			} else {
				prompt = fmt.Sprintf("В выгрузке %d уникальных пользователей. Каждый представлен %d строками. После SELECT DISTINCT user_id сколько строк останется?", n, m)
				answer = fmt.Sprint(n)
			}
		case "analytics-middle":
			if family == 1 {
				prompt = fmt.Sprintf("У %d клиентов по %d заказа и по 2 тикета поддержки. Обе таблицы соединены только по client_id. Сколько строк даёт INNER JOIN заказов с тикетами?", n, m)
				answer = fmt.Sprint(n * m * 2)
			} else {
				prompt = fmt.Sprintf("В 7-дневный retention входят %d пользователей зрелой когорты. Ещё %d зарегистрировались 2 дня назад. Других ограничений нет. Каков знаменатель D7 при исключении незрелых пользователей?", n*100, m*100)
				answer = fmt.Sprint(n * 100)
			}
		case "analytics-senior":
			if family == 1 {
				prompt = fmt.Sprintf("До очистки %d наблюдений. %d не имеют целевой переменной, ещё %d других содержат признаки из будущего. Группы не пересекаются; обе исключаются. Сколько наблюдений останется?", n*100, m, m+2)
				answer = fmt.Sprint(n*100 - 2*m - 2)
			} else {
				prompt = fmt.Sprintf("В %d источниках по %d событий с разными event_id. Полный повтор доставки каждого события добавил ещё одну копию. После глобальной дедупликации сколько уникальных событий останется?", n, m)
				answer = fmt.Sprint(n * m)
			}
		}
	} else {
		switch spec + "-" + grade {
		case "backend-junior":
			if family == 1 {
				prompt = fmt.Sprintf("Обработчик последовательно выполняет %d вызовов по %d мс и ещё 10 мс вычислений. Каково общее время в мс?", n, m)
				answer = fmt.Sprint(n*m + 10)
			} else {
				prompt = fmt.Sprintf("%d независимых запросов запущены одновременно, каждый длится %d мс. Ограничений параллелизма и других задержек нет. Через сколько мс завершатся все запросы?", n, m*10)
				answer = fmt.Sprint(m * 10)
			}
		case "backend-middle":
			if family == 1 {
				prompt = fmt.Sprintf("Пул содержит %d соединений. Каждый запрос занимает соединение ровно 100 мс. При непрерывной занятости и без других ограничений какова максимальная пропускная способность в запросах/с?", n)
				answer = fmt.Sprint(n * 10)
			} else {
				prompt = fmt.Sprintf("В среднем приходит %d запросов/с, среднее время в системе %d мс, режим стационарный. По закону Литтла каково среднее число запросов в системе? Округлите до двух знаков.", n*100, m*10)
				answer = fmt.Sprint(n * m)
			}
		case "backend-senior":
			if family == 1 {
				prompt = fmt.Sprintf("Пик %d запросов/с. При полной загрузке одна реплика выдерживает 100 запросов/с. Целевая загрузка не выше 50%%. Сколько реплик нужно минимум? Округлите вверх до целого.", n*100+m*10)
				answer = fmt.Sprint(int(math.Ceil(float64(n*100+m*10) / 50)))
			} else {
				prompt = fmt.Sprintf("Есть %d реплик по 100 запросов/с. После отказа одной оставшиеся держат целевую загрузку не выше 50%%. Какую нагрузку в запросах/с они выдержат в рамках этой цели?", n)
				answer = fmt.Sprint((n - 1) * 50)
			}
		case "frontend-junior":
			if family == 1 {
				prompt = fmt.Sprintf("Контейнер шириной %d px без padding и border. В нём две одинаковые колонки и gap 20 px. Какова ширина каждой колонки в px?", n*100)
				answer = fmt.Sprint((n*100 - 20) / 2)
			} else {
				prompt = fmt.Sprintf("В массиве %d элементов. После filter осталось %d, затем map удвоил каждое значение. Сколько элементов в результате?", n+m, n)
				answer = fmt.Sprint(n)
			}
		case "frontend-middle":
			if family == 1 {
				prompt = fmt.Sprintf("Начальный пакет %d КБ. Из него выделили %d КБ в ленивый модуль, нужный только при открытии отчёта. До открытия отчёта сколько КБ JS нужно загрузить?", n*100, m*10)
				answer = fmt.Sprint(n*100 - m*10)
			} else {
				prompt = fmt.Sprintf("Виртуальный список: viewport %d px, строка 50 px, scroll выровнен по строке, overscan %d строки с каждой стороны. В середине списка сколько строк в DOM?", n*50, m)
				answer = fmt.Sprint(n + 2*m)
			}
		case "frontend-senior":
			if family == 1 {
				prompt = fmt.Sprintf("Синхронный JS занимает %d мс. Целевой бюджет задачи 50 мс. На сколько мс задача превышает бюджет?", n*50)
				answer = fmt.Sprint(n*50 - 50)
			} else {
				prompt = fmt.Sprintf("На загрузку %d КБ при скорости 100 КБ/с уходило время передачи. Новый пакет на %d КБ меньше. На сколько мс сократится передача? Сжатие и задержки игнорируются.", n*100, m*10)
				answer = fmt.Sprint(m * 100)
			}
		case "analytics-junior":
			if family == 1 {
				prompt = fmt.Sprintf("За три дня число заказов: %d, %d, %d. Каково среднее дневное число заказов?", n, n+m, n+2*m)
				answer = fmt.Sprint(n + m)
			} else {
				prompt = fmt.Sprintf("Отсортированный ряд из 5 значений: %d, %d, %d, %d, %d. Какова медиана?", n, n+m, n+2*m, n+3*m, n+4*m)
				answer = fmt.Sprint(n + 2*m)
			}
		case "analytics-middle":
			if family == 1 {
				prompt = fmt.Sprintf("В контрольной группе %d00 пользователей и конверсия 10%%. В тестовой столько же пользователей и конверсия 12%%. Сколько дополнительных конверсий в тестовой группе?", n)
				answer = fmt.Sprint(n * 2)
			} else {
				prompt = fmt.Sprintf("%d заказов при выручке %d ₽. Каков средний чек в рублях?", n, n*m*100)
				answer = fmt.Sprint(m * 100)
			}
		case "analytics-senior":
			if family == 1 {
				prompt = fmt.Sprintf("При независимых одинаково распределённых наблюдениях стандартная ошибка оценки равна %d. Размер выборки вырос в 4 раза, дисперсия не меняется. Какова новая стандартная ошибка?", n*2)
				answer = fmt.Sprint(n)
			} else {
				prompt = fmt.Sprintf("Семейная вероятность ошибки первого рода задана 0,05. Применяем поправку Бонферрони для %d проверок. Каков порог для одной проверки в процентах? Округлите до двух знаков.", m)
				answer = fmt.Sprintf("%.2f", 5.0/float64(m))
			}
		}
	}
	return PrivateQuestion{Question: Question{Prompt: prompt, Topic: topic, Kind: "number", Options: []string{}}, Expected: answer}
}
