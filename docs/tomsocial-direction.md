# TomSocial direction

TomSocial is a possible later social layer for TomChat. It should reuse locally held TomChat device identity and reviewed cryptography rather than introducing a second identity system.

Public posts and profile updates are signed objects exchanged directly between known peers. Each user's devices retain local copies. Users may explicitly replicate selected public content to their own trusted devices. There is no shared relay, global index or central feed database.

Global searchable discovery is not promised. Public content visibility, moderation, retention, synchronisation and abuse handling require separate product and threat-model decisions before implementation.
