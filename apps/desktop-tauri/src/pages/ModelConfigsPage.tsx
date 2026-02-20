import { FormEvent, useEffect, useState } from "react";

import { createModelConfig, listModelConfigs } from "../api/runtimeClient";

export function ModelConfigsPage() {
  const [provider, setProvider] = useState("openai");
  const [model, setModel] = useState("gpt-4.1-mini");
  const [secretRef, setSecretRef] = useState("keychain:openai:default");
  const [modelConfigs, setModelConfigs] = useState<Array<Record<string, string>>>([]);

  const refresh = async () => {
    const payload = await listModelConfigs();
    setModelConfigs(payload.model_configs);
  };

  useEffect(() => {
    void refresh();
  }, []);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    await createModelConfig({ provider, model, secret_ref: secretRef });
    await refresh();
  };

  return (
    <section className="panel">
      <h2>Model Configs</h2>
      <form className="form-grid" onSubmit={onSubmit}>
        <label>
          Provider
          <select value={provider} onChange={(event) => setProvider(event.target.value)}>
            <option value="openai">openai</option>
            <option value="anthropic">anthropic</option>
          </select>
        </label>
        <label>
          Model
          <input value={model} onChange={(event) => setModel(event.target.value)} />
        </label>
        <label>
          Secret Ref
          <input value={secretRef} onChange={(event) => setSecretRef(event.target.value)} />
        </label>
        <button type="submit">Save Model Config</button>
      </form>
      <ul>
        {modelConfigs.map((item) => (
          <li key={item.model_config_id}>
            {item.provider}:{item.model} ({item.secret_ref})
          </li>
        ))}
      </ul>
    </section>
  );
}
