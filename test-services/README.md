# Test Services Suite

This directory contains standalone Go microservices designed to test **Triage's** panic interception, monorepo AST extraction, AI root-cause diagnosis, and automated bugfix PR generation across realistic backend architectures.

---

## Services Overview

| Service             | Directory                       | Default Port | Simulation Types                                                                                      |
| :------------------ | :------------------------------ | :----------- | :---------------------------------------------------------------------------------------------------- |
| **Simple Service**  | `test-services/simple-service`  | `:8081`      | Basic nil pointer dereference                                                                         |
| **Order Service**   | `test-services/order-service`   | `:8082`      | Multi-package domain logic, nested nil struct pointers, slice bounds out of range, uninitialized maps |
| **Payment Gateway** | `test-services/payment-gateway` | `:8083`      | Interface method panics, uninitialized config struct dereference, integer divide-by-zero              |
| **Auth Service**    | `test-services/auth-service`    | `:8084`      | Deep nested nil claims dereference, short header slice bounds out of range, send to closed channel    |

---

## Project Onboarding in Triage

To track any of these test microservices in Triage:

1. Open your Triage Console at **http://localhost:8080** (or **http://localhost:3000** during local development).
2. Click **"New Project"** (`+`).
3. Select or enter your repository: `algotyrnt/triage` (or your repository).
4. Set the **Monorepo Subdirectory (`root_dir`)**:
   - For **Order Service**: `test-services/order-service`
   - For **Payment Gateway**: `test-services/payment-gateway`
   - For **Auth Service**: `test-services/auth-service`
   - For **Simple Service**: `test-services/simple-service`
5. Click **Initialize Project**, copy the **Ingestion API Key** displayed on the completion screen, and store it securely.

---

## Quickstart & Environment Setup

Each service includes a `.env.example` file. Copy it to `.env.local` or `.env` inside the respective service directory:

```bash
# Example: configure Order Service
cd test-services/order-service
cp .env.example .env.local
```

### Environment Variables

| Variable            | Required? | Default                                  | Description                                         |
| :------------------ | :-------- | :--------------------------------------- | :-------------------------------------------------- |
| `TRIAGE_API_KEY`    | **Yes**   | —                                        | Project-scoped ingestion key from Triage Onboarding |
| `TRIAGE_ENGINE_URL` | No        | `http://localhost:8080/api/v1/telemetry` | Telemetry endpoint on the Triage engine             |
| `PORT`              | No        | Service-specific (`8081`-`8084`)         | Port the HTTP service listens on                    |

---

## Running with `-trimpath` (Recommended)

When running or building Go services with the Triage SDK, provide `TRIAGE_API_KEY` and the `-trimpath` flag:

```bash
# 1. Order Service (:8082)
cd test-services/order-service
TRIAGE_API_KEY=your_sample_api_key go run -trimpath .

# 2. Payment Gateway (:8083)
cd test-services/payment-gateway
TRIAGE_API_KEY=your_sample_api_key go run -trimpath .

# 3. Auth Service (:8084)
cd test-services/auth-service
TRIAGE_API_KEY=your_sample_api_key go run -trimpath .

# 4. Simple Service (:8081)
cd test-services/simple-service
TRIAGE_API_KEY=your_sample_api_key go run -trimpath .
```

Or when compiling a production binary:

```bash
go build -trimpath -o server .
./server
```

### 💡 Why `-trimpath` Matters

- **Without `-trimpath`:** Go's compiler bakes your local machine's absolute file path into runtime stack traces (e.g. `/Users/yourusername/projects/triage/test-services/order-service/pkg/orders/service.go:35`).
- **With `-trimpath`:** Go strips all local filesystem prefixes and produces clean, module-relative paths (e.g. `pkg/orders/service.go:35` or `order-service/pkg/orders/service.go:35`).
- **Production Parity:** Production binaries (Docker, CI/CD, Kubernetes) are always compiled with `-trimpath`. Using `-trimpath` during local simulation ensures runtime stack traces match production formatting, enabling Triage's AST engine to seamlessly map file paths to repository source trees.

---

## Panic Simulation Trigger Endpoints

### 1. Simple Service (`:8081`)

- **Nil Pointer Panic:**
  ```bash
  curl http://localhost:8081/crash
  ```

### 2. Order Service (`:8082`)

- **Nested Nil Struct Pointer** (`order.Customer.ShippingAddress.ZipCode`):
  ```bash
  curl http://localhost:8082/orders/checkout-nil-address
  ```
- **Slice Index Out of Range** (`rules[0]` on empty slice in `calculator.go`):
  ```bash
  curl http://localhost:8082/orders/discount-empty-slice
  ```
- **Assignment to Nil Map** (`order.Metadata[key] = val` in `service.go`):
  ```bash
  curl http://localhost:8082/orders/uninitialized-metadata
  ```

### 3. Payment Gateway (`:8083`)

- **Nil Interface Method Call** (`p.VaultClient.TokenizeCard()`):
  ```bash
  curl http://localhost:8083/payments/nil-vault-client
  ```
- **Uninitialized Config Pointer** (`p.MerchantConfig.WebhookSecret`):
  ```bash
  curl http://localhost:8083/payments/nil-config
  ```
- **Integer Division by Zero** (`totalFee / installments` when `installments == 0` in `fees/calculator.go`):
  ```bash
  curl http://localhost:8083/payments/zero-division
  ```

### 4. Auth & Identity Service (`:8084`)

- **Deep Nested Nil Pointer** (`session.Claims.User.Profile.Email` in `service.go`):
  ```bash
  curl http://localhost:8084/auth/profile-nil-claims
  ```
- **Slice Bounds on Short String** (`authHeader[7:]` when header length < 7 in `helpers.go`):
  ```bash
  curl http://localhost:8084/auth/token-slice-bounds
  ```
- **Send on Closed Channel** (`ch <- msg` in `service.go`):
  ```bash
  curl http://localhost:8084/auth/closed-channel
  ```
