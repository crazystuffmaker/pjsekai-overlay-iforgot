package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/go-github/v57/github"
	"github.com/crazystuffmaker/pjsekai-overlay-iforgot/pkg/pjsekaioverlay"
	"github.com/srinathh/gokilo/rawmode"
	"golang.org/x/sys/windows"
)

func shouldCheckUpdate() bool {
	executablePath, err := os.Executable()
	if err != nil {
		return false
	}
	updateCheckFile, err := os.OpenFile(filepath.Join(filepath.Dir(executablePath), ".update-check"), os.O_RDONLY, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		return false
	}
	defer updateCheckFile.Close()

	scanner := bufio.NewScanner(updateCheckFile)
	scanner.Scan()
	lastCheckTime, err := strconv.ParseInt(scanner.Text(), 10, 64)
	if err != nil {
		return false
	}

	return time.Now().Unix()-lastCheckTime > 60*60*24
}

func checkUpdate() {
	githubClient := github.NewClient(nil)
	release, _, err := githubClient.Repositories.GetLatestRelease(context.Background(), "crazystuffmaker", "pjsekai-overlay-iforgot")
	if err != nil {
		return
	}

	executablePath, err := os.Executable()
	updateCheckFile, err := os.OpenFile(filepath.Join(filepath.Dir(executablePath), ".update-check"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return
	}
	defer updateCheckFile.Close()
	updateCheckFile.WriteString(strconv.FormatInt(time.Now().Unix(), 10))

	latestVersion := strings.TrimPrefix(release.GetTagName(), "v")
	if latestVersion == pjsekaioverlay.Version {
		return
	}
	fmt.Printf("New pjsekai-overlay version: v%s -> v%s\n", pjsekaioverlay.Version, latestVersion)
	fmt.Printf("Download link: %s\n", release.GetHTMLURL())
}

func origMain(isOptionSpecified bool) {
	Title()

	var skipAviutlInstall bool
	flag.BoolVar(&skipAviutlInstall, "no-aviutl-install", false, "i forgot")

	var outDir string
	flag.StringVar(&outDir, "out-dir", "./dist/_chartId_", "idk what the translation would be :c")

	var teamPower int
	flag.IntVar(&teamPower, "team-power", 250000, "Specify team power.")

	var apCombo bool
	flag.BoolVar(&apCombo, "ap-combo", true, "Enables AP effect for the combo.")

	flag.Usage = func() {
		fmt.Println("Usage: pjsekai-overlay [Chart ID] [オプション]")
		flag.PrintDefaults()
	}

	flag.Parse()

	if shouldCheckUpdate() {
		checkUpdate()
	}

	if !skipAviutlInstall {
		success := pjsekaioverlay.TryInstallObject()
		if success {
			fmt.Println("idk what this does so no translation for it :p")
		}
	}

	var chartId string
	if flag.Arg(0) != "" {
		chartId = flag.Arg(0)
		fmt.Printf("Chart ID: %s\n", color.GreenString(chartId))
	} else {
		fmt.Print("Enter the chart ID 'sekai-best-' and 'chcy-' (Sekai Best charts do not work yet). \n-> ")
		fmt.Scanln(&chartId)
		fmt.Printf("\033[A\033[2K\r> %s\n", color.GreenString(chartId))
	}

	chartSource, err := pjsekaioverlay.DetectChartSource(chartId)
	if err != nil {
		fmt.Println(color.RedString("The server for the chart could not be found. Please enter the correct chart ID, including the prefix.."))
		return
	}
	fmt.Printf("%s%s%s Downloading chart sheet... ", RgbColorEscape(chartSource.Color), chartSource.Name, ResetEscape())
	chart, err := pjsekaioverlay.FetchChart(chartSource, chartId)

	if err != nil {
		fmt.Println(color.RedString(fmt.Sprintf("Fail: %s", err.Error())))
		return
	}
	if chart.Engine.Version != 12 {
		fmt.Println(color.RedString(fmt.Sprintf("Fail: The engine is not supported! (version %d）", chart.Engine.Version)))
		return
	}

	fmt.Println(color.GreenString("Success"))
	fmt.Printf("  %s / %s - %s (Lv. %s)\n",
		color.CyanString(chart.Title),
		color.CyanString(chart.Artists),
		color.CyanString(chart.Author),
		color.MagentaString(strconv.Itoa(chart.Rating)),
	)

	fmt.Printf("Getting exe path... ")
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println(color.RedString(fmt.Sprintf("Fail: %s", err.Error())))
		return
	}

	fmt.Println(color.GreenString("Success"))

	cwd, err := os.Getwd()

	if err != nil {
		fmt.Println(color.RedString(fmt.Sprintf("Fail: %s", err.Error())))
		return
	}

	formattedOutDir := filepath.Join(cwd, strings.Replace(outDir, "_chartId_", chartId, -1))
	fmt.Printf("Output directory: %s\n", color.CyanString(filepath.Dir(formattedOutDir)))

	fmt.Print("Downloading cover image... ")
	err = pjsekaioverlay.DownloadCover(chartSource, chart, formattedOutDir)
	if err != nil {
		fmt.Println(color.RedString(fmt.Sprintf("Fail: %s", err.Error())))
		return
	}

	fmt.Println(color.GreenString("Success"))

	fmt.Print("Downloading background image... ")
	err = pjsekaioverlay.DownloadBackground(chartSource, chart, formattedOutDir)
	if err != nil {
		fmt.Println(color.RedString(fmt.Sprintf("Fail: %s", err.Error())))
		return
	}

	fmt.Println(color.GreenString("Success"))

	fmt.Print("Reading chart... ")
	levelData, err := pjsekaioverlay.FetchLevelData(chartSource, chart)

	if err != nil {
		fmt.Println(color.RedString(fmt.Sprintf("Fail: %s", err.Error())))
		return
	}

	fmt.Println(color.GreenString("Success"))

	if !isOptionSpecified {
		fmt.Print("Please specify your team power.\n-> ")
		var tmpTeamPower string
		fmt.Scanln(&tmpTeamPower)
		teamPower, err = strconv.Atoi(tmpTeamPower)
		if err != nil {
			fmt.Println(color.RedString(fmt.Sprintf("Fail: %s", err.Error())))
			return
		}
		fmt.Printf("\033[A\033[2K\r> %s\n", color.GreenString(tmpTeamPower))

	}

	fmt.Print("Calculating score... ")
	scoreData := pjsekaioverlay.CalculateScore(chart, levelData, teamPower)

	fmt.Println(color.GreenString("Success"))

	if !isOptionSpecified {
		fmt.Print("Enable AP combo？ (Y/n)\n-> ")
		before, _ := rawmode.Enable()
		tmpEnableComboApByte, _ := bufio.NewReader(os.Stdin).ReadByte()
		tmpEnableComboAp := string(tmpEnableComboApByte)
		rawmode.Restore(before)
		fmt.Printf("\n\033[A\033[2K\r> %s\n", color.GreenString(tmpEnableComboAp))
		if tmpEnableComboAp == "Y" || tmpEnableComboAp == "y" || tmpEnableComboAp == "" {
			apCombo = true
		} else {
			apCombo = false
		}
	}
	executableDir := filepath.Dir(executablePath)
	assets := filepath.Join(executableDir, "assets")

	fmt.Print("Generating ped file... ")

	err = pjsekaioverlay.WritePedFile(scoreData, assets, apCombo, filepath.Join(formattedOutDir, "data.ped"))

	if err != nil {
		fmt.Println(color.RedString(fmt.Sprintf("Failed: %s", err.Error())))
		return
	}

	fmt.Println(color.GreenString("Success"))

	fmt.Print("Generating exo file... ")

	composerAndVocals := []string{chart.Artists, "？"}
	if separateAttempt := strings.Split(chart.Artists, " / "); chartSource.Id == "chart_cyanvas" && len(separateAttempt) <= 2 {
		composerAndVocals = separateAttempt
	}

	artists := fmt.Sprintf("作詞：ー    作曲：%s    編曲：ー\r\nVo：%s   ", composerAndVocals[0], composerAndVocals[1], chart.Author)

	err = pjsekaioverlay.WriteExoFiles(assets, formattedOutDir, chart.Title, artists)

	if err != nil {
		fmt.Println(color.RedString(fmt.Sprintf("Failed: %s", err.Error())))
		return
	}

	fmt.Println(color.GreenString("Success"))

	fmt.Println(color.GreenString("\nAll processing done, import the exo into AviUtl.(located in pjsekai-overlay/dist/chcy-'chartid')"))
}

func main() {
	isOptionSpecified := len(os.Args) > 1
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32

	windows.GetConsoleMode(stdout, &originalMode)
	windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	origMain(isOptionSpecified)

	if !isOptionSpecified {
		fmt.Print(color.CyanString("\nPress any key to exit.."))

		before, _ := rawmode.Enable()
		bufio.NewReader(os.Stdin).ReadByte()
		rawmode.Restore(before)
	}
}
