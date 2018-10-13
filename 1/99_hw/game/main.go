package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Room struct {
	RoomsAvailable []string
	Description    string
	WelcomePhrase  string
}

type ClosedRoom struct {
	Room
	Closed bool
}

type Bedroom struct {
	Room
	Equipment map[string]Equipment
	OnTable   []string
	OnChair   []string
}

type Kitchen struct {
	Room
	Equipment map[string]Equipment
	Tasks     []string
	OnTable   []string
}

type User struct {
	CurrentRoom        RoomInterface
	Equipment          []string
	EquipmentActivated bool
}

type Equipment struct {
	name string
}

type Keys struct {
	Equipment
	Object string
	Result string
	Room   string
}

type ClothesEquipment struct {
	Equipment
}

type TaskPackBackpack struct{}

type TaskGoToUniversity struct{}

type TaskInterface interface {
	complited() bool
}

type EquipmentInterface interface {
	use(string) string
	putOn() string
}

type RoomInterface interface {
	isRoomAvailable(string) bool
	lookAround() string
	comeIn() (bool, string)
	takeSmth(string) string
	putOnSmth(string) string
	open()
}

const (
	initialRoom = "кухня"
)

var (
	rooms         = map[string]RoomInterface{}
	equipmentList = map[string]EquipmentInterface{}
	tasksList     = map[string]TaskInterface{}
	user          = User{}
)

func (r *Room) isRoomAvailable(room string) bool {
	for _, item := range r.RoomsAvailable {
		if item == room {
			return true
		}
	}

	return false
}

func (u *User) changeRoom(room string) string {
	if u.CurrentRoom.isRoomAvailable(room) {
		isOpen, welcomePhrase := rooms[room].comeIn()
		if isOpen {
			u.CurrentRoom = rooms[room]
		}
		return welcomePhrase
	}

	return "нет пути в " + room
}

func (k *Kitchen) lookAround() string {
	res := k.Description + ", "

	if len(k.OnTable) != 0 {
		res += "на столе: " + strings.Join(k.OnTable, ", ") + ", "
	}

	if len(k.Tasks) != 0 {
		for idx, item := range k.Tasks {
			if tasksList[item].complited() == true {
				k.Tasks = append(k.Tasks[:idx], k.Tasks[idx+1:]...)
			}
		}
		res += "надо " + strings.Join(k.Tasks, " и ") + ". "
	}

	return res + "можно пройти - " + strings.Join(k.RoomsAvailable, ", ")
}

func (b *Bedroom) lookAround() string {
	return func() (res string) {
		res = b.Description
		str := []string{}

		if len(b.OnTable) != 0 {
			str = append(str, "на столе: "+strings.Join(b.OnTable, ", "))
		}

		if len(b.OnChair) != 0 {
			str = append(str, "на стуле: "+strings.Join(b.OnChair, ", "))
		}

		if len(str) != 0 {
			res = strings.Join(str, ", ")
		}

		res += ". "

		return
	}() + "можно пройти - " + strings.Join(b.RoomsAvailable, ", ")
}

func (r *Room) lookAround() string {
	return r.Description + ". " + "можно пройти - " +
		strings.Join(r.RoomsAvailable, ", ")
}

func (r *Room) comeIn() (bool, string) {
	return true, r.WelcomePhrase + ". " + "можно пройти - " +
		strings.Join(r.RoomsAvailable, ", ")
}

func (r *Room) open() {}

func (r *ClosedRoom) comeIn() (bool, string) {
	if r.Closed == true {
		return false, "дверь закрыта"
	}

	return true, r.WelcomePhrase + ". " + "можно пройти - " +
		strings.Join(r.RoomsAvailable, ", ")
}

func (r *ClosedRoom) open() {
	r.Closed = false
}

func (k *Kitchen) takeSmth(smth string) string {
	for idx, item := range k.OnTable {
		if smth == item {
			if user.EquipmentActivated == false {
				return "некуда класть"
			}
			k.OnTable = append(k.OnTable[:idx], k.OnTable[idx+1:]...)
			return user.addToEquipment(smth)
		}
	}

	return "нет такого"
}

func (b *Bedroom) takeSmth(smth string) string {
	if user.EquipmentActivated == false {
		return "некуда класть"
	}

	for idx, item := range b.OnTable {
		if smth == item {
			b.OnTable = append(b.OnTable[:idx], b.OnTable[idx+1:]...)
			return user.addToEquipment(smth)
		}
	}

	for idx, item := range b.OnChair {
		if smth == item {
			b.OnChair = append(b.OnChair[:idx], b.OnChair[idx+1:]...)
			return user.addToEquipment(smth)
		}
	}

	return "нет такого"
}

func (b *Bedroom) putOnSmth(smth string) string {
	for idx, item := range b.OnTable {
		if smth == item {
			b.OnTable = append(b.OnTable[:idx], b.OnTable[idx+1:]...)
			return equipmentList[smth].putOn()
		}
	}

	for idx, item := range b.OnChair {
		if smth == item {
			b.OnChair = append(b.OnChair[:idx], b.OnChair[idx+1:]...)
			return equipmentList[smth].putOn()
		}
	}

	return "нет такого"
}

func (t *TaskPackBackpack) complited() bool {
	for _, item := range user.Equipment {
		if item == "рюкзак" {
			return true
		}
	}

	return false
}

func (t *TaskGoToUniversity) complited() bool {
	return false
}

func (u *User) addToEquipment(smth string) string {
	u.Equipment = append(u.Equipment, smth)
	return "предмет добавлен в инвентарь: " + smth
}

func (u *User) lookAround() string {
	return u.CurrentRoom.lookAround()
}

func (u *User) takeSmth(smth string) string {
	return u.CurrentRoom.takeSmth(smth)
}

func (u *User) putOnSmth(smth string) string {
	return u.CurrentRoom.putOnSmth(smth)
}

func (u *User) use(smth string, obj string) string {
	for _, item := range u.Equipment {
		if item == smth {
			return equipmentList[item].use(obj)
		}
	}

	return "нет предмета в инвентаре - " + smth
}

func (e *Equipment) use(obj string) string {
	return "не к чему применить"
}

func (e *Keys) use(obj string) string {
	if e.Object == obj {
		rooms[e.Room].open()
		return e.Result
	}

	return "не к чему применить"
}

func (e *Equipment) putOn() string {
	return "нельзя надеть - " + e.name
}

func (e *ClothesEquipment) putOn() string {
	if e.name == "рюкзак" {
		user.EquipmentActivated = true
	}

	user.addToEquipment(e.name)

	return "вы надели: " + e.name
}

func (r *Room) takeSmth(smth string) string {
	return "нет такого"
}

func (r *Room) putOnSmth(smth string) string {
	return "нет такого"
}

func initGame() {
	rooms = map[string]RoomInterface{
		"кухня": &Kitchen{
			Room: Room{
				RoomsAvailable: []string{"коридор"},
				Description:    "ты находишься на кухне",
				WelcomePhrase:  "кухня, ничего интересного",
			},
			OnTable: []string{"чай"},
			Tasks: []string{
				"собрать рюкзак",
				"идти в универ",
			},
		},

		"коридор": &Room{
			RoomsAvailable: []string{"кухня", "комната", "улица"},
			Description:    "пустая комната",
			WelcomePhrase:  "ничего интересного",
		},

		"улица": &ClosedRoom{
			Closed: true,
			Room: Room{
				Description:    "на улице весна",
				WelcomePhrase:  "на улице весна",
				RoomsAvailable: []string{"домой"},
			},
		},

		"комната": &Bedroom{
			Room: Room{
				Description:    "пустая комната",
				WelcomePhrase:  "ты в своей комнате",
				RoomsAvailable: []string{"коридор"},
			},
			OnTable: []string{"ключи", "конспекты"},
			OnChair: []string{"рюкзак"},
		},
	}

	equipmentList = map[string]EquipmentInterface{
		"ключи": &Keys{
			Equipment: Equipment{
				name: "ключи",
			},
			Object: "дверь",
			Result: "дверь открыта",
			Room:   "улица",
		},
		"чай": &Equipment{
			name: "чай",
		},
		"конспекты": &Equipment{
			name: "конспекты",
		},
		"рюкзак": &ClothesEquipment{
			Equipment{
				name: "рюкзак",
			},
		},
	}

	tasksList = map[string]TaskInterface{
		"собрать рюкзак": &TaskPackBackpack{},
		"идти в универ":  &TaskGoToUniversity{},
	}

	user = User{
		CurrentRoom:        rooms[initialRoom],
		EquipmentActivated: false,
		Equipment:          []string{},
	}
}

func handleCommand(command string) string {
	action := strings.Split(command, " ")

	switch action[0] {
	case "осмотреться":
		return user.lookAround()
	case "идти":
		return user.changeRoom(action[1])
	case "взять":
		return user.takeSmth(action[1])
	case "надеть":
		return user.putOnSmth(action[1])
	case "применить":
		return user.use(action[1], action[2])
	default:
		return "неизвестная команда"
	}
}

func main() {
	initGame()

	in := bufio.NewScanner(os.Stdin)

	for in.Scan() {
		line := in.Text()
		fmt.Println(handleCommand(line))
	}
}
