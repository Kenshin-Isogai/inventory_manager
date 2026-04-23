@echo off
powershell -ExecutionPolicy Bypass -File "%~dp0run-cloud-migrations.ps1" %*
