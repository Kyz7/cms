@echo off
REM Script untuk build Docker image CMS (Windows)

set IMAGE_NAME=cms-integrasi
set IMAGE_TAG=%1
if "%IMAGE_TAG%"=="" set IMAGE_TAG=latest

echo 🐳 Building Docker image: %IMAGE_NAME%:%IMAGE_TAG%

REM Build Docker image
docker build -t %IMAGE_NAME%:%IMAGE_TAG% .

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ Build completed successfully!
    echo.
    echo 📦 Image: %IMAGE_NAME%:%IMAGE_TAG%
    echo.
    echo 🚀 To run the container:
    echo    docker run -p 8080:8080 --env-file .env %IMAGE_NAME%:%IMAGE_TAG%
    echo.
    echo 🧪 To test locally:
    echo    docker run -p 8080:8080 -e DB_HOST=localhost -e DB_NAME=starpi -e DB_USER=postgres -e DB_PASSWORD=your_password -e JWT_SECRET=your_jwt_secret_minimum_32_characters_long %IMAGE_NAME%:%IMAGE_TAG%
) else (
    echo ❌ Build failed!
    exit /b 1
)

