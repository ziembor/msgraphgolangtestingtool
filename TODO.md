\# TODO



\- -thumbprint code do not work, failing with message 


2026/01/03 14:01:38 Authentication setup failed: failed to export cert from store: powershell export failed: exit status 1, output: Export-PfxCertificate : Cannot find drive. A drive with the name 'cert' does not exist.

At line:1 char:163

\+ ... Char($\_) }; Export-PfxCertificate -Cert cert:\\CurrentUser\\My\\cd817b33 ...

\+                 ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

&nbsp;   + CategoryInfo          : ObjectNotFound: (cert:String) \[Export-PfxCertificate], DriveNotFoundException

&nbsp;   + FullyQualifiedErrorId : DriveNotFound,Microsoft.CertificateServices.Commands.ExportPfxCertificate


- END

