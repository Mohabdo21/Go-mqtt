// This is a Milesight AM300 decoder for the Milesight AM300 series of sensors.
function decode(hexPayload) {
  const buffer = hexToBytes(hexPayload);
  const xorKey = 0x5a;
  const result = {
    Temperature: 0,
    Humidity: 0,
    CO2: 0,
  };

  for (let i = 0; i < buffer.length; i++) {
    buffer[i] ^= xorKey;
  }

  for (let i = 0; i < buffer.length - 3; i += 4) {
    const channel = buffer[i];
    const type = buffer[i + 1];
    const value = (buffer[i + 2] << 8) | buffer[i + 3];

    if (channel === 0x03 && type === 0x67) {
      result.Temperature = value / 10.0;
    } else if (channel === 0x04 && type === 0x68) {
      result.Humidity = value / 2.0;
    } else if (channel === 0x07 && type === 0x7d) {
      result.CO2 = value;
    }
  }

  return result;
}

function hexToBytes(hex) {
  const bytes = [];
  for (let c = 0; c < hex.length; c += 2) {
    bytes.push(parseInt(hex.substr(c, 2), 16));
  }
  return new Uint8Array(bytes);
}
