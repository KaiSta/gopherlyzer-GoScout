@echo off

for /f %%f in ('dir /b "."') do (
  if not %%f == instr.bat (
    go run ..\main.go -in %%f -out ..\results2\%%f -link
  )
)