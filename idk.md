# Project Introduction

Universal Android Debloater TUI is a terminal-based Android debloating tool built using ADB and a modern TUI framework. It provides a lightweight, keyboard-driven interface to manage and remove pre-installed or unwanted Android system applications without requiring a graphical interface or external GUI tools. The project focuses on simplicity, speed, and developer-friendly interaction through the terminal.

Unlike traditional debloaters that rely on executable GUI applications, this tool runs entirely inside the terminal, making it portable, minimal, and efficient for advanced users who are familiar with ADB debugging.

## Project Objective
The main objectives of this project are:
- To create a lightweight terminal-based Android debloating tool
- To eliminate the need for bulky GUI debloater applications
- To provide a fast and keyboard-driven ADB interface
- To simplify the process of listing, selecting, and removing Android packages
- To build a clean and extensible TUI system using Go and Bubble Tea
- To improve productivity for developers and advanced Android users

## Key Features (Planned / Implemented)
- ADB device detection and connection status
- Interactive terminal UI (TUI)
- List installed Android packages via ADB
- Select/unselect apps using keyboard navigation
- Batch uninstall selected packages
- Minimal Catppuccin-themed UI design
- Screen-based navigation system (Home → ADB → Debloat)
- Lightweight and dependency-minimal architecture

## Project Scope
This project is designed for:
- Advanced Android users
- Developers working with ADB and system optimization
- Users who prefer terminal-based tools over GUI applications
- Lightweight system management on Linux/macOS/Windows

It is not intended for beginners and assumes basic knowledge of ADB and Android debugging.

## Tools & Technologies Used
- Programming Language: Go (Golang)
- TUI Framework: Bubble Tea (Charmbracelet)
- Styling Library: Lip Gloss
- Android Interface: ADB (Android Debug Bridge)

## Expected Outcome
The final application will provide a smooth and efficient terminal experience for managing Android applications. It will allow users to quickly debloat devices without relying on external GUI tools, improving speed, control, and accessibility for power users.