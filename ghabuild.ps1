# ghabuild.ps1
# GitHub Actions Build Preparation Script
# Verifies branch name, commits changes, pushes to main, creates and pushes tag

Write-Host "GitHub Actions Build Preparation" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Read VERSION file
if (-not (Test-Path "VERSION")) {
    Write-Host "ERROR: VERSION file not found!" -ForegroundColor Red
    Write-Host "The VERSION file must exist in the project root." -ForegroundColor Yellow
    exit 1
}

$version = (Get-Content VERSION).Trim()
Write-Host "Version from VERSION file: $version" -ForegroundColor Green

# Step 2: Get current git branch name
try {
    $currentBranch = git rev-parse --abbrev-ref HEAD 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Failed to get current git branch!" -ForegroundColor Red
        Write-Host "Make sure you're in a git repository." -ForegroundColor Yellow
        exit 1
    }
    Write-Host "Current git branch: $currentBranch" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Git command failed: $_" -ForegroundColor Red
    exit 1
}

# Step 3: Verify branch name matches version pattern
$expectedBranch = "b$version"
Write-Host ""
Write-Host "Verification:" -ForegroundColor Cyan
Write-Host "  Expected branch: $expectedBranch" -ForegroundColor White
Write-Host "  Current branch:  $currentBranch" -ForegroundColor White

if ($currentBranch -ne $expectedBranch) {
    Write-Host ""
    Write-Host "ERROR: Branch name does not match VERSION file!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Current branch is:  $currentBranch" -ForegroundColor Yellow
    Write-Host "Expected branch is: $expectedBranch" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Please do one of the following:" -ForegroundColor White
    Write-Host "  1. Checkout the correct branch: git checkout $expectedBranch" -ForegroundColor Cyan
    Write-Host "  2. Create and checkout the branch: git checkout -b $expectedBranch" -ForegroundColor Cyan
    Write-Host "  3. Update the VERSION file to match current branch" -ForegroundColor Cyan
    Write-Host ""
    exit 1
}

Write-Host ""
Write-Host "SUCCESS: Branch name matches VERSION file!" -ForegroundColor Green
Write-Host ""

# Step 4: Check for uncommitted changes
Write-Host "Checking for uncommitted changes..." -ForegroundColor Cyan
$gitStatus = git status --porcelain
if ([string]::IsNullOrWhiteSpace($gitStatus)) {
    Write-Host "No uncommitted changes found." -ForegroundColor Yellow
    Write-Host ""
} else {
    Write-Host "Uncommitted changes found:" -ForegroundColor Yellow
    Write-Host $gitStatus
    Write-Host ""
}

# Step 5: Get commit message from user
$commitMessage = Read-Host "Enter commit message (press Enter to skip commit)"

if ([string]::IsNullOrWhiteSpace($commitMessage)) {
    Write-Host "Skipping commit (no message provided)." -ForegroundColor Yellow
} else {
    Write-Host ""
    Write-Host "Committing all changes..." -ForegroundColor Cyan
    git commit -a -m $commitMessage
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Git commit failed!" -ForegroundColor Red
        exit 1
    }
    Write-Host "Commit successful!" -ForegroundColor Green
}

# Step 6: Push current branch to main
Write-Host ""
Write-Host "Pushing branch '$currentBranch' to main..." -ForegroundColor Cyan
$pushConfirm = Read-Host "Do you want to push '$currentBranch' to main? (y/n)"
if ($pushConfirm -ne 'y') {
    Write-Host "Push to main cancelled by user." -ForegroundColor Yellow
    exit 0
}

git checkout main 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to checkout main branch!" -ForegroundColor Red
    exit 1
}

git merge $currentBranch
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to merge '$currentBranch' into main!" -ForegroundColor Red
    git checkout $currentBranch 2>&1 | Out-Null
    exit 1
}

git push origin main
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to push main to origin!" -ForegroundColor Red
    git checkout $currentBranch 2>&1 | Out-Null
    exit 1
}
Write-Host "Successfully pushed to main!" -ForegroundColor Green

# Switch back to version branch
git checkout $currentBranch 2>&1 | Out-Null

# Step 7: Create and push git tag
Write-Host ""
$tagName = "v$version"
Write-Host "Creating git tag: $tagName..." -ForegroundColor Cyan

# Check if tag already exists
$existingTag = git tag -l $tagName
if ($existingTag) {
    Write-Host "WARNING: Tag '$tagName' already exists locally!" -ForegroundColor Yellow
    $tagConfirm = Read-Host "Do you want to delete and recreate it? (y/n)"
    if ($tagConfirm -eq 'y') {
        git tag -d $tagName
        Write-Host "Deleted existing local tag." -ForegroundColor Yellow
    } else {
        Write-Host "Tag creation cancelled." -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Script completed!" -ForegroundColor Green
        exit 0
    }
}

git tag $tagName
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to create tag!" -ForegroundColor Red
    exit 1
}
Write-Host "Tag '$tagName' created successfully!" -ForegroundColor Green

# Step 8: Push tag to GitHub
Write-Host ""
Write-Host "Pushing tag '$tagName' to GitHub..." -ForegroundColor Cyan
Write-Host "WARNING: This will trigger GitHub Actions workflow!" -ForegroundColor Yellow
$tagPushConfirm = Read-Host "Do you want to push the tag? (y/n)"
if ($tagPushConfirm -ne 'y') {
    Write-Host "Tag push cancelled by user." -ForegroundColor Yellow
    Write-Host "To push later, run: git push origin $tagName" -ForegroundColor Cyan
    exit 0
}

git push origin $tagName
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to push tag to GitHub!" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "SUCCESS! All operations completed!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Summary:" -ForegroundColor Cyan
Write-Host "  Version: $version" -ForegroundColor White
Write-Host "  Branch: $currentBranch" -ForegroundColor White
Write-Host "  Tag: $tagName" -ForegroundColor White
Write-Host "  Merged to: main" -ForegroundColor White
Write-Host "  Tag pushed to: GitHub" -ForegroundColor White
Write-Host ""
Write-Host "GitHub Actions workflow should now be triggered!" -ForegroundColor Yellow
Write-Host "Check: gh run list --limit 5" -ForegroundColor Cyan
Write-Host ""

exit 0
