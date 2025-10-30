package main

import (
	"ADB-CLI/config"
	"ADB-CLI/models"
	"fmt"
	"os"
	"os/exec"
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
	default:
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

// ✅ Refactor FillTemplate — agar otomatis replace semua key dari map
func FillTemplate(template string, values map[string]string) string {
	for key, val := range values {
		template = strings.ReplaceAll(template, "{"+key+"}", val)
	}
	return template
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

// ✅ Refactor runADBCommand agar otomatis menangani {scrcpyPath}
func runADBCommand(command string, devices []models.DeviceType) error {
	conf, err := config.LoadJSON[models.AdbConfigType]("config/commandADB.json")
	if err != nil {
		fmt.Println("Failed to load config from json", err)
		return fmt.Errorf("failed to load config from json: %v", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no devices found")
	}

	// Kill & Start adb server
	RunCommand(strings.ReplaceAll(conf.AdbCommandTemplate["kill"], "{adbPath}", conf.AdbPath))
	time.Sleep(3 * time.Second)
	RunCommand(strings.ReplaceAll(conf.AdbCommandTemplate["start"], "{adbPath}", conf.AdbPath))
	time.Sleep(3 * time.Second)

	for _, device := range devices {
		values := map[string]string{
			"adbPath":     conf.AdbPath,
			"scrcpyPath":  conf.ScrcpyPath, // ✅ tambahkan support untuk scrcpyPath
			"ip":          device.IPAddress,
			"port":        fmt.Sprintf("%d", conf.AdbPort),
		}

		// Hubungkan device dulu
		connectOutput, errCon := RunCommand(FillTemplate(conf.AdbCommandTemplate["connect"], values))
		if errCon != nil {
			fmt.Printf("❌ Failed to connect to %s (%s): %v\n", device.Name, device.IPAddress, errCon)
			fmt.Printf("↪️  Output: %s\n", connectOutput)
			continue
		}

		// Jalankan command (bisa berisi {scrcpyPath} atau {adbPath})
		output, err := RunCommand(FillTemplate(command, values))
		if err != nil {
			fmt.Printf("❌ Error executing command on %s (%s): %v\n", device.Name, device.IPAddress, err)
			continue
		}

		fmt.Printf("✅ Output from %s (%s):\n%s\n", device.Name, device.IPAddress, output)
	}

	return nil
}
