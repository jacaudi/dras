package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(INFO, &buf)

	// Should log INFO and above
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	
	// DEBUG should not be logged
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG message should not be logged when level is INFO")
	}
	
	// INFO, WARN, ERROR should be logged
	if !strings.Contains(output, "info message") {
		t.Error("INFO message should be logged")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN message should be logged")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message should be logged")
	}
}

func TestLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(INFO, &buf)
	
	// Change to DEBUG level
	logger.SetLevel(DEBUG)
	logger.Debug("debug message")
	
	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("DEBUG message should be logged after setting level to DEBUG")
	}
}

func TestParseLevel(t *testing.T) {
	testCases := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"WARNING", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"FATAL", FATAL},
		{"fatal", FATAL},
		{"invalid", INFO}, // default
	}
	
	for _, tc := range testCases {
		result := ParseLevel(tc.input)
		if result != tc.expected {
			t.Errorf("ParseLevel(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestLogger_MessageFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(INFO, &buf)
	
	logger.Info("test message")
	
	output := buf.String()
	
	// Check for timestamp format
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Error("Output should contain timestamp in brackets")
	}
	
	// Check for level
	if !strings.Contains(output, "INFO:") {
		t.Error("Output should contain level indicator")
	}
	
	// Check for message
	if !strings.Contains(output, "test message") {
		t.Error("Output should contain the actual message")
	}
}

func TestFieldLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(INFO, &buf)
	
	fieldLogger := logger.WithField("station", "KATX")
	fieldLogger.Info("radar data fetched")
	
	output := buf.String()
	
	if !strings.Contains(output, "radar data fetched") {
		t.Error("Output should contain the message")
	}
	
	if !strings.Contains(output, "station=KATX") {
		t.Error("Output should contain the field")
	}
}

func TestFieldLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithOutput(INFO, &buf)
	
	fields := map[string]string{
		"station": "KATX",
		"vcp":     "R31",
	}
	
	fieldLogger := logger.WithFields(fields)
	fieldLogger.Info("processing station")
	
	output := buf.String()
	
	if !strings.Contains(output, "processing station") {
		t.Error("Output should contain the message")
	}
	
	if !strings.Contains(output, "station=KATX") {
		t.Error("Output should contain station field")
	}
	
	if !strings.Contains(output, "vcp=R31") {
		t.Error("Output should contain vcp field")
	}
}

func TestDefaultLogger(t *testing.T) {
	// Test that default logger functions work
	// We can't easily capture output from the default logger in tests
	// since it writes to os.Stdout, but we can verify they don't panic
	
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Default logger functions should not panic: %v", r)
		}
	}()
	
	// These should not panic
	Debug("debug message")
	Info("info message") 
	Warn("warn message")
	Error("error message")
	
	WithField("key", "value").Info("field message")
	WithFields(map[string]string{"key": "value"}).Info("fields message")
}

func TestLogger_Fatal(t *testing.T) {
	// We can't easily test Fatal since it calls os.Exit(1)
	// This would be better tested with integration tests
	// For now, just verify the method exists and is callable
	var buf bytes.Buffer
	logger := NewWithOutput(FATAL+1, &buf) // Set level higher than FATAL so it won't actually exit
	
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Fatal with high level should not panic: %v", r)
		}
	}()
	
	logger.Fatal("this should not exit due to level")
	
	// Should not have written anything due to level filtering
	if buf.Len() > 0 {
		t.Error("Fatal should not log when level is too high")
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	testCases := []struct {
		loggerLevel Level
		messageLevel Level
		shouldLog    bool
	}{
		{DEBUG, DEBUG, true},
		{DEBUG, INFO, true},
		{DEBUG, WARN, true},
		{DEBUG, ERROR, true},
		{DEBUG, FATAL, true},
		{INFO, DEBUG, false},
		{INFO, INFO, true},
		{INFO, WARN, true},
		{INFO, ERROR, true},
		{INFO, FATAL, true},
		{ERROR, DEBUG, false},
		{ERROR, INFO, false},
		{ERROR, WARN, false},
		{ERROR, ERROR, true},
		{ERROR, FATAL, true},
	}
	
	for _, tc := range testCases {
		var buf bytes.Buffer
		logger := NewWithOutput(tc.loggerLevel, &buf)
		
		switch tc.messageLevel {
		case DEBUG:
			logger.Debug("test message")
		case INFO:
			logger.Info("test message")
		case WARN:
			logger.Warn("test message")
		case ERROR:
			logger.Error("test message")
		case FATAL:
			// Don't actually call Fatal as it exits
			continue
		}
		
		hasOutput := buf.Len() > 0
		if hasOutput != tc.shouldLog {
			t.Errorf("Logger level %v, message level %v: expected shouldLog=%v, got hasOutput=%v",
				tc.loggerLevel, tc.messageLevel, tc.shouldLog, hasOutput)
		}
	}
}