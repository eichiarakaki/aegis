
### TODO
- [x] Normalize the responses
- [x] Make the orchestrator
- [x] Proper handling of 'stop' session
- [x] Make the health checks funtionalities
- [x] Register component when attached instead of registering when executed
- [x] Normalize the "data": {} structure for EVERY response to the aegisctl
- [ ] Implement BackPressure to the NATS server
- [ ] Develop the realtime data streaming
- [ ] Implement an installation for the configuration file to user's local folder such as ~/.config/aegis/config.yaml

- [x] Implement a 'aegisctl session start [session] —from [timestamp/datetime] —to [timestamp/datetime]'
  - taking into consideration the time differences
- [x] Implement 'aegisctl session restart [session]'
- [x] Implement 'aegisctl session resume [session]' from a stopped session

- [x] Improve the aegisctl's formatter for better context when attaching components

- [ ] Adapt the SDKs for restarting (cleaning all previous data)