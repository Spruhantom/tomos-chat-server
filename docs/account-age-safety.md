# Account age and child safety

Age-band preview, age assurance and guardian consent are separate concepts. A date-of-birth entry or local age calculation is not proof of age and must not unlock restricted features by itself.

Before child-facing production release:
- minimise personal data and avoid retaining full date of birth unless strictly required;
- prevent profile edits from bypassing age restrictions;
- enforce safeguards in domain logic and on every client;
- implement authenticated, revocable guardian-device pairing and consent;
- restrict discovery, groups and contact approval appropriately;
- provide block/report controls and age-appropriate defaults;
- complete privacy, safeguarding and applicable legal reviews for each launch jurisdiction.

Any age-assurance design must be reconciled with the self-hosted deployment model and documented data flows. This document is a requirements baseline, not evidence that age assurance or guardian consent is implemented.
