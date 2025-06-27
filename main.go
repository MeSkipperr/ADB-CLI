package main

import (
	"ADB-CLI/config"
	"ADB-CLI/models"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/AlecAivazis/survey/v2"
)

func clearTerminal() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default: // Linux/macOS
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func RunCommand(command string) (string, error) {
	args := strings.Fields(command)

	if len(args) == 0 {
		return "", fmt.Errorf("command is empty")
	}

	cmd := exec.Command(args[0], args[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command error: %v\nOutput: %s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}

func FillTemplate(template string, values map[string]string) string {
	for key, val := range values {
		template = strings.ReplaceAll(template, "{"+key+"}", val)
	}
	return template
}

func selectRoomNumber() ([]models.DeviceType, bool) {
	clearTerminal()

	message := `
============================
TV Control Terminal 

Through this terminal, you can access and control TV devices by room number using ADB commands.
============================
`
	fmt.Println(message)
	validRoom := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	var roomNumbers []string
	foundRooms := make(map[string]bool)
	var typeRoomSelect string

	typeRoomSelectPromtn := &survey.Select{
		Message: "Do you want to target one/multiple rooms or all rooms?",
		Options: []string{"Single or Multiple Rooms", "All Rooms"},
	}

	err := survey.AskOne(typeRoomSelectPromtn, &typeRoomSelect)
	if err != nil {
		fmt.Println("Error selecting option:", err)
		return nil, true
	}

	if typeRoomSelect == "Single or Multiple Rooms" {
		var roomStringValue string

		info := `
		📌 Enter room number(s):
		• For a single room: e.g., 1001 or 1001B
		• For multiple rooms: separate them with commas, e.g., 1001, 1002A, 1003B
		`

		fmt.Println(info)

		survey.AskOne(&survey.Input{
			Message: "Enter Room Number(s):",
		}, &roomStringValue)

		raw := strings.Split(roomStringValue, ",")

		for _, r := range raw {
			room := strings.TrimSpace(r)
			if room != "" {
				if !validRoom.MatchString(room) {
					fmt.Printf("❌ Invalid room format: %s (only letters and numbers allowed)\n", room)
					return nil, true
				}
				roomNumbers = append(roomNumbers, room)
			}
		}

		if len(roomNumbers) == 0 {
			fmt.Println("❌ No valid room numbers provided. Please try again.")
			return nil, true
		}

		db, err := sql.Open("sqlite", "file:./resource/app.db")
		if err != nil {
			fmt.Printf("❌ Error opening database: %v\n", err)
			panic(err)
		}
		defer db.Close()

		placeholders := make([]string, len(roomNumbers))
		args := make([]interface{}, len(roomNumbers))

		for i, v := range roomNumbers {
			placeholders[i] = "?"
			args[i] = fmt.Sprintf("IPTV Room %s", v)
		}

		query := fmt.Sprintf("SELECT * FROM rooms WHERE type = Android TV & name IN (%s)", strings.Join(placeholders, ","))

		rows, err := db.Query(query, args...)
		if err != nil {
			fmt.Printf("❌ Error querying database: %v\n", err)
			return nil, true
		}
		defer rows.Close()

		devices := []models.DeviceType{}

		for rows.Next() {
			var d models.DeviceType
			err := rows.Scan(
				&d.ID,
				&d.Name,
				&d.IPAddress,
				&d.Device,
				&d.Error,
				&d.Description,
				&d.DownTime,
				&d.Type,
			)
			if err != nil {
				panic(err)
			}
			devices = append(devices, d)

			for _, rn := range roomNumbers {
				if d.Name == fmt.Sprintf("IPTV Room %s", rn) {
					foundRooms[rn] = true
					break
				}
			}
		}

		fmt.Println("✅ Rooms found in the database:")

		found := false
		for _, rn := range roomNumbers {
			if foundRooms[rn] {
				fmt.Println("-", rn)
				found = true
			}
		}

		if !found {
			fmt.Println("⚠️ No matching rooms found in the database.")
			return nil, true
		}

		return devices, false
	} else {
		db, err := sql.Open("sqlite", "file:./resource/app.db")
		if err != nil {
			fmt.Printf("❌ Error opening database: %v\n", err)
			panic(err)
		}
		defer db.Close()

		query := "SELECT * FROM rooms WHERE type = Android TV"

		rows, err := db.Query(query)
		if err != nil {
			fmt.Printf("❌ Error querying database: %v\n", err)
			return nil, true
		}
		defer rows.Close()

		devices := []models.DeviceType{}

		for rows.Next() {
			var d models.DeviceType
			err := rows.Scan(
				&d.ID,
				&d.Name,
				&d.IPAddress,
				&d.Device,
				&d.Error,
				&d.Description,
				&d.DownTime,
				&d.Type,
			)
			if err != nil {
				panic(err)
			}
			devices = append(devices, d)
		}
		return devices, false
	}
}

func getADBCommand() (bool, string) {
	conf, errJSON := config.LoadJSON[models.AdbConfigType]("config/commandADB.json")
	if errJSON != nil {
		fmt.Println("Failed to load config from json", errJSON)
		return true, "Failed to load config from json: " + errJSON.Error()
	}

	options := []string{"Custom Command"}
	for _, cmd := range conf.ADBListCommand {
		options = append(options, cmd.Title)
	}

	commandPromnt := &survey.Select{
		Message: "Select ADB Command :",
		Options: options,
	}
	var commandSelect string

	err := survey.AskOne(commandPromnt, &commandSelect)
	if err != nil {
		fmt.Println("Error selecting command:", err)
		return true, "Error selecting command: " + err.Error()
	}

	var selectedIndex int
	for i, opt := range options {
		if opt == commandSelect {
			selectedIndex = i
			break
		}
	}

	if commandSelect == "Custom Command" {
		var customCommand string
		info := `
		📌 Please construct your custom command using the following placeholders:
		• Format of custom command: {adbPath} -s {ip}:{port} shell <your_command>
		• Example : {adbPath} -s {ip}:{port} shell cat /proc/uptime
		`

		fmt.Println(info)
		survey.AskOne(&survey.Input{
			Message: "Enter your custom ADB command:",
		}, &customCommand)
		if customCommand == "" {
			fmt.Println("❌ Custom command cannot be empty.")
			return true, "Custom command cannot be empty."
		}
		if !strings.Contains(customCommand, "{adbPath}") || !strings.Contains(customCommand, "{ip}") || !strings.Contains(customCommand, "{port}") {
			fmt.Println("❌ Custom command must contain {adbPath}, {ip}, and {port}.")
			return true, "Custom command must contain {adbPath}, {ip}, and {port}."
		}
		return false, customCommand
	}
	return false, conf.ADBListCommand[selectedIndex-1].Command
}

func runADBCommand(command string, devices []models.DeviceType) (error) {
	conf, err := config.LoadJSON[models.AdbConfigType]("config/adb.json")
	if err != nil {
		fmt.Println("Failed to load config from json", err)
		return fmt.Errorf("failed to load config from json: %v", err)
	}
	adbPath := conf.AdbPath

	if len(devices) == 0 {
		return fmt.Errorf("no devices found")
	}

	RunCommand(strings.ReplaceAll(conf.AdbCommandTemplate["kill"], "{adbPath}", adbPath))
	time.Sleep(5 * time.Second) // Wait for 5 seconds to ensure the ADB server is killed
	RunCommand(strings.ReplaceAll(conf.AdbCommandTemplate["start"], "{adbPath}", adbPath))
	time.Sleep(5 * time.Second) // Wait for 5 seconds to ensure the ADB server is started


	for _, device := range devices {
		if device.Error {
			continue
		}
		cmd := FillTemplate(command, map[string]string{
			"adbPath": conf.AdbPath,
			"ip":      device.IPAddress,
			"port":    fmt.Sprintf("%d", conf.AdbPort),
		})

		output, err := RunCommand(cmd)
		if err != nil {
			fmt.Printf("Error on %s: %v", device.Name, err)
			continue
		}
		fmt.Printf("Output from %s: %s", device.Name, output)
	}

	return nil
}

func main() {
	for {
		var repeat string
		roomNumber, err := selectRoomNumber()

		if roomNumber == nil || err {
			fmt.Println("❌ Exiting due to invalid input.")
			survey.AskOne(&survey.Select{
				Message: "Do you want to run the tool again?",
				Options: []string{"Yes", "No"},
			}, &repeat)
			return
		}

		err, value := getADBCommand()
		if err {
			fmt.Println("❌ Error running ADB command:", value)
			survey.AskOne(&survey.Select{
				Message: "Do you want to run the tool again?",
				Options: []string{"Yes", "No"},
			}, &repeat)
			return
		}

		fmt.Println(value)

		errRunADB := runADBCommand(value, roomNumber )
		if errRunADB != nil {
			fmt.Println("❌ Error running ADB command:", err)
		}

		survey.AskOne(&survey.Select{
			Message: "Do you want to run the tool again?",
			Options: []string{"Yes", "No"},
		}, &repeat)

		if repeat == "No" {
			fmt.Println("👋 Exiting. Thank you!")
			break
		}
	}
}
