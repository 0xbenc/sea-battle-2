@echo off
setlocal
"%~dp0sea-battle.exe" %*
set "game_exit=%errorlevel%"
if not "%game_exit%"=="0" (
    echo.
    echo Sea Battle 2 exited with error %game_exit%.
    pause
)
exit /b %game_exit%
