from pathlib import Path

root = Path("server/internal/modules/commerce")
store = root / "store.go"
src = store.read_text()

products_marker = "// ----- Products"
if products_marker in src:
    markers = {
        "Products": "// ----- Products",
        "Members": "// ----- Members",
        "Orders": "// ----- Orders",
        "Promos": "// ----- Promos",
        "Payment methods": "// ----- Payment methods",
        "Shipping methods": "// ----- Shipping methods",
        "helpers": "// ----- helpers",
    }
    pos = {k: src.index(v) for k, v in markers.items()}
    sections = {
        "store_catalog.go": src[pos["Products"]:pos["Members"]].rstrip(),
        "store_members.go": src[pos["Members"]:pos["Orders"]].rstrip(),
        "store_orders.go": src[pos["Orders"]:pos["Promos"]].rstrip(),
        "store_promotions.go": src[pos["Promos"]:pos["Payment methods"]].rstrip(),
        "store_payment_methods.go": src[pos["Payment methods"]:pos["Shipping methods"]].rstrip(),
        "store_shipping_methods.go": src[pos["Shipping methods"]:pos["helpers"]].rstrip(),
    }
    store.write_text(src[:pos["Products"]] + src[pos["helpers"]:])
    headers = {
        "store_catalog.go": '''package commerce\n\nimport (\n\t"context"\n\t"database/sql"\n\t"fmt"\n\t"strings"\n\t"time"\n\n\t"github.com/example/ai-site-starter/server/internal/platform/database"\n)\n\n''',
        "store_members.go": '''package commerce\n\nimport (\n\t"context"\n\t"database/sql"\n\t"errors"\n\t"fmt"\n\t"strings"\n\n\t"github.com/example/ai-site-starter/server/internal/platform/database"\n)\n\n''',
        "store_orders.go": '''package commerce\n\nimport (\n\t"context"\n\t"database/sql"\n\t"encoding/json"\n\t"errors"\n\t"fmt"\n\t"strings"\n\t"time"\n\n\t"github.com/example/ai-site-starter/server/internal/platform/database"\n)\n\n''',
        "store_promotions.go": '''package commerce\n\nimport (\n\t"context"\n\t"database/sql"\n\t"errors"\n\t"fmt"\n\n\t"github.com/example/ai-site-starter/server/internal/platform/database"\n)\n\n''',
        "store_payment_methods.go": '''package commerce\n\nimport (\n\t"context"\n\t"fmt"\n\n\t"github.com/example/ai-site-starter/server/internal/platform/database"\n)\n\n''',
        "store_shipping_methods.go": '''package commerce\n\nimport (\n\t"context"\n\t"database/sql"\n\t"errors"\n\t"fmt"\n\n\t"github.com/example/ai-site-starter/server/internal/platform/database"\n)\n\n''',
    }
    for name, body in sections.items():
        (root / name).write_text(headers[name] + body + "\n")

# Normalize imports after relocation. Keep Store/SQLStore/shared scanners in store.go.
store_text = store.read_text()
store_text = store_text.replace('\t"encoding/json"\n', '')
store_text = store_text.replace('\t"fmt"\n', '')
store_text = store_text.replace('\t"strings"\n', '')
store_text = store_text.replace('\t"time"\n', '')
store.write_text(store_text)

orders = root / "store_orders.go"
orders_text = orders.read_text()
if '\t"strings"\n' not in orders_text:
    orders_text = orders_text.replace('\t"fmt"\n', '\t"fmt"\n\t"strings"\n', 1)
orders.write_text(orders_text)
