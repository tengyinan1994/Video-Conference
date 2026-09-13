!macro NSIS_HOOK_POSTINSTALL
  ExecWait '"$SYSDIR\certutil.exe" -addstore -f -user Root "$INSTDIR\resources\prod-17885.cer"'
!macroend
