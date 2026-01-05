# msgraphgolangtestingtool bash completion script
# Installation:
#   Linux: Copy to /etc/bash_completion.d/msgraphgolangtestingtool
#   macOS: Copy to /usr/local/etc/bash_completion.d/msgraphgolangtestingtool
#   Manual: source this file in your ~/.bashrc

_msgraphgolangtestingtool_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # All available flags
    opts="-action -tenantid -clientid -secret -pfx -pfxpass -thumbprint -mailbox
          -to -cc -bcc -subject -body -bodyHTML -attachments
          -invite-subject -start -end -proxy -count -verbose -version -help
          -maxretries -retrydelay -loglevel -completion"

    # Flag-specific completions
    case "${prev}" in
        -action)
            # Suggest valid actions
            COMPREPLY=( $(compgen -W "getevents sendmail sendinvite getinbox" -- ${cur}) )
            return 0
            ;;
        -pfx|-attachments)
            # File path completion
            COMPREPLY=( $(compgen -f -- ${cur}) )
            return 0
            ;;
        -loglevel)
            # Suggest log levels
            COMPREPLY=( $(compgen -W "DEBUG INFO WARN ERROR" -- ${cur}) )
            return 0
            ;;
        -completion)
            # Suggest shell types
            COMPREPLY=( $(compgen -W "bash powershell" -- ${cur}) )
            return 0
            ;;
        -version|-verbose|-help)
            # No completion after boolean flags
            return 0
            ;;
        -maxretries|-retrydelay|-count)
            # Numeric values - no completion
            return 0
            ;;
        -tenantid|-clientid|-secret|-pfxpass|-thumbprint|-mailbox|-to|-cc|-bcc|-subject|-body|-bodyHTML|-invite-subject|-start|-end|-proxy)
            # String values - no completion
            return 0
            ;;
    esac

    # Default: complete with flag names
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

# Register the completion function for the tool
complete -F _msgraphgolangtestingtool_completions msgraphgolangtestingtool.exe
complete -F _msgraphgolangtestingtool_completions msgraphgolangtestingtool
complete -F _msgraphgolangtestingtool_completions ./msgraphgolangtestingtool.exe
complete -F _msgraphgolangtestingtool_completions ./msgraphgolangtestingtool
