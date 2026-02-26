package tools

func RegisterBaseTools(registry *Registry) error {
	if err := RegisterCoreTools(registry); err != nil {
		return err
	}
	if err := registry.Register(NewEchoTool()); err != nil {
		return err
	}
	if err := registry.Register(NewRunCommandTool()); err != nil {
		return err
	}
	return nil
}
