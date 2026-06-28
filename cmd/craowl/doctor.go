package main

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/royhairul/craowl/internal/browser"
	"github.com/royhairul/craowl/internal/browser/recorder"
	"github.com/spf13/cobra"
)

func runDoctorBrowser(cmd *cobra.Command, args []string) error {
	fmt.Println("Running Forensic Browser Diagnostics...")
	fmt.Println("Initializing Phase 1 Foundation & Phase 2 Recorders...")

	// 1. Initialize foundation
	manager := browser.GetBrowserManager()
	opts := browser.DefaultBrowserOptions()
	opts.Headless = true

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session, err := manager.StartSession(ctx, "doctor_profile", opts)
	if err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}
	defer manager.EndSession("doctor_profile")

	tab := session.NewTab()
	defer tab.Cancel()

	// 2. Initialize and start recorders
	netRec := recorder.NewNetworkRecorder()
	conRec := recorder.NewConsoleRecorder()
	excRec := recorder.NewExceptionRecorder()
	lifeRec := recorder.NewLifecycleRecorder()
	snapRec := recorder.NewScreenshotRecorder()

	recorders := []recorder.Recorder{netRec, conRec, excRec, lifeRec, snapRec}

	for _, rec := range recorders {
		if err := rec.Start(tab.Ctx); err != nil {
			fmt.Printf("Warning: failed to start a recorder: %v\n", err)
		}
	}

	// 3. Navigate
	targetUrl := "https://shopee.co.id/verify/traffic/error"
	if len(args) > 0 {
		targetUrl = args[0]
	}
	fmt.Printf("Navigating to %s...\n", targetUrl)

	err = chromedp.Run(tab.Ctx,
		chromedp.Navigate(targetUrl),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		fmt.Printf("Navigation error: %v\n", err)
	}

	// 4. Stop and Export
	for _, rec := range recorders {
		rec.Stop()
	}

	netData, _ := netRec.Export()
	conData, _ := conRec.Export()

	fmt.Println("\n--- DIAGNOSTIC RESULTS ---")

	reqs, ok := netData.([]recorder.NetworkRequest)
	if ok {
		fmt.Printf("Network Requests: %d\n", len(reqs))
		for _, r := range reqs {
			fmt.Printf(" - [%d] %s %s\n", r.Status, r.Method, r.URL)
		}
	}

	msgs, ok := conData.([]recorder.ConsoleMessage)
	if ok {
		fmt.Printf("\nConsole Logs: %d\n", len(msgs))
		for _, m := range msgs {
			fmt.Printf(" - [%s] %v\n", m.Type, m.Args)
		}
	}

	fmt.Println("\nDiagnostics completed successfully.")
	return nil
}
