# Account age and child safety

Age-band preview, age assurance and guardian consent are separate concepts. A date-of-birth entry or local age calculation is not proof of age and must not unlock restricted features by itself.

Before child-facing production release:
- minimise personal data and avoid retaining full date of birth unless strictly required;
- prevent profile edits from bypassing age restrictions;
- enforce safeguards in shared domain logic and every client;
- implement authenticated, revocable guardian-device pairing and consent;
- restrict discovery, groups and contact approval appropriately;
- provide block/report controls and age-appropriate defaults;
- complete privacy, safeguarding and applicable legal reviews for each launch jurisdiction.

Age and consent data should remain local or be exchanged directly between explicitly authorised devices, with clear user consent and documented data flows. This document is a requirements baseline, not evidence that age assurance or guardian consent is implemented.
