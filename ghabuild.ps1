 [string]$versionStr =Get-Content .\src\VERSION
 [string]$branch = 'b{0}' -f (Get-Content  .\src\VERSION)
[string]$changelogPath  = ('Changelog\{0}.md' -f (Get-Content  .\src\VERSION))
if(test-path $changelogPath) {
    [string]$commit = Get-Content   ('Changelog\{0}.md' -f (Get-Content  .\src\VERSION))
}
else {
    "## [{0}] {1:yyy-MM-dd}`n`n### Added`n`n- lets's try" -f $versionStr,(get-date) | Out-File -Path $changelogPath -Encoding utf8
    [string]$commit = Get-Content   $changelogPath
} 
$tag = 'v{0}' -f (Get-Content  .\src\VERSION)
git checkout -b $branch 
git status
git commit -a -m $commit
git push origin $branch 
git tag $tag ; git push origin $tag 
gh pr create --title $branch --body $commit  ; gh pr merge --delete-branch
git status
git pull origin main