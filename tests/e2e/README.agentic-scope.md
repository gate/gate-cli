# Synthetic scope contract regression

`scope_contract.py` contains independent permission expectations and isolated, resource-bound credential fixtures. It does not implement a gateway or claim to validate a remote authorization policy.

```sh
# Pure contract, environment-isolation and temporary-profile assertions.
python3 -B -m unittest discover -s tests/e2e -p "test_*.py" -v
# Real CLI and SDK requests through synthetic local HTTP peers.
make test-cli-e2e
```

These checks retain the twelve Bot operations, eight stop variants and the existing flash operation permissions. Internal service integration is maintained as a separate suite; public checks do not replace its authorization or business-forwarding assertions.
