from pathlib import Path

root = Path("server/internal/modules/commerce")
path = root / "service.go"
src = path.read_text()

orders = src.index("// ----- Orders")
quote = src.index("// QuoteInput is the browser-supplied quote request.", orders)
status = src.index("// UpdateOrderStatus advances an order through the state machine", quote)
returns = src.index("// UpdateOrderReturnStatus advances the return request state machine", status)
restock = src.index("// RestockOrder was removed", returns)
members = src.index("// ----- Members", restock)

order_lookup = src[orders:quote].rstrip()
checkout = src[quote:status].rstrip()
order_status = src[status:returns].rstrip()
return_status = src[returns:restock].rstrip()
restock_body = src[restock:members].rstrip()

# Remove the contiguous Orders section from the central service file and
# normalize imports whose only consumer moved with that section.
central = src[:orders] + src[members:]
central = central.replace('\t"net/mail"\n', '')
path.write_text(central)

(root / "service_orders.go").write_text('''package commerce\n\nimport (\n\t"context"\n\t"fmt"\n\t"strings"\n\t"time"\n\n\t"github.com/example/ai-site-starter/server/internal/auth"\n)\n\n''' + order_lookup + "\n\n" + order_status + "\n")

(root / "service_checkout.go").write_text('''package commerce\n\nimport (\n\t"context"\n\t"errors"\n\t"fmt"\n\t"net/mail"\n\t"strings"\n\t"time"\n\n\t"github.com/example/ai-site-starter/server/internal/auth"\n)\n\n''' + checkout + "\n")

(root / "service_returns.go").write_text('''package commerce\n\nimport (\n\t"context"\n\t"strings"\n\t"time"\n\n\t"github.com/example/ai-site-starter/server/internal/auth"\n)\n\n''' + return_status + "\n")

(root / "service_restock.go").write_text('''package commerce\n\nimport (\n\t"context"\n\t"crypto/sha256"\n\t"encoding/hex"\n\t"encoding/json"\n\t"errors"\n\t"fmt"\n\t"strings"\n\t"time"\n\n\t"github.com/example/ai-site-starter/server/internal/auth"\n)\n\n''' + restock_body + "\n")
