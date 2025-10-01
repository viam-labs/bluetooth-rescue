# bluetooth-rescue 

The bluetooth system on the raspberry pi 4b intermittently fails while tethering. This module detects the failure and attempts to repair the system by restarting the relevant bits.

## Model viam:bluetooth-rescue:rescue

Detect and optionally repair bluetooth crashes.

### Configuration

The following attribute template can be used to configure this model:

```json
{
  "rescue": <bool>
}
```

#### Attributes

The following attributes are available for this model:

| Name          | Type   | Inclusion | Description                |
|---------------|--------|-----------|----------------------------|
| `rescue` | bool   | Optional  | If true, the module will rescue bluetooth by restarting it, instead of just logging errors |

#### Example Configuration

```json
{
  "rescue": true
}
```

### DoCommand

#### Example DoCommand

```json
{
  "action": "rescue"
}
```
