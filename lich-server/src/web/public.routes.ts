import type { FastifyPluginAsync } from 'fastify';
import fs from 'node:fs';
import path from 'node:path';

const popStyles = `
  @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:ital,wght@0,400;0,700;0,800;1,400&family=Space+Grotesk:wght@500;700;800&display=swap');

  :root {
    --bg: #0d0b18;
    --surface: #18142a;
    --surface-clay: #211c38;
    --border: #352f55;
    --primary: #a855f7;
    --pop-pink: #ff2a85;
    --pop-yellow: #facc15;
    --pop-cyan: #06b6d4;
    --pop-green: #22c55e;
    --pop-orange: #f97316;
    --text: #e2e8f0;
    --text-heading: #ffffff;
    --muted: #94a3b8;
  }

  * { box-sizing: border-box; margin: 0; padding: 0; }
  
  body {
    font-family: 'Space Grotesk', -apple-system, sans-serif;
    background-color: var(--bg);
    color: var(--text);
    line-height: 1.6;
    background-image: 
      radial-gradient(circle at 15% 15%, rgba(255, 42, 133, 0.15) 0%, transparent 40%),
      radial-gradient(circle at 85% 25%, rgba(6, 182, 212, 0.18) 0%, transparent 45%),
      radial-gradient(circle at 50% 85%, rgba(168, 85, 247, 0.15) 0%, transparent 50%);
    background-attachment: fixed;
    overflow-x: hidden;
  }

  .container {
    max-width: 960px;
    margin: 0 auto;
    padding: 32px 24px 80px;
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 24px;
    background: var(--surface);
    border: 2px solid var(--border);
    border-radius: 20px;
    box-shadow: 0 6px 0 var(--border), 0 12px 24px rgba(0,0,0,0.3);
    margin-bottom: 48px;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 12px;
    font-weight: 800;
    font-size: 1.4rem;
    color: var(--text-heading);
    text-decoration: none;
  }

  .logo-badge {
    background: linear-gradient(135deg, var(--pop-pink), var(--primary));
    color: #fff;
    padding: 6px 14px;
    border-radius: 12px;
    font-family: 'JetBrains Mono', monospace;
    font-weight: 800;
    font-size: 0.95rem;
    box-shadow: 0 3px 0 #7e22ce;
    transform: rotate(-2deg);
    display: inline-block;
  }

  nav a {
    color: var(--text);
    text-decoration: none;
    margin-left: 20px;
    font-weight: 700;
    font-size: 0.95rem;
    transition: all 0.2s;
    padding: 6px 12px;
    border-radius: 8px;
  }
  nav a:hover {
    color: var(--pop-yellow);
    background: rgba(255,255,255,0.05);
  }

  .hero {
    text-align: center;
    margin-bottom: 40px;
  }

  .hero-tag {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    background: var(--surface-clay);
    border: 2px solid var(--border);
    padding: 8px 18px;
    border-radius: 999px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.9rem;
    color: var(--pop-yellow);
    font-weight: 700;
    box-shadow: 0 4px 0 var(--border);
    margin-bottom: 24px;
  }

  h1 {
    font-size: 3.2rem;
    font-weight: 800;
    line-height: 1.15;
    margin-bottom: 20px;
    color: #ffffff;
    letter-spacing: -1px;
  }

  .gradient-text {
    background: linear-gradient(135deg, var(--pop-pink), var(--pop-yellow), var(--pop-cyan));
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .tagline {
    font-size: 1.3rem;
    color: var(--muted);
    max-width: 680px;
    margin: 0 auto 32px;
    font-weight: 500;
  }

  /* Mascot Showcase */
  .mascot-container {
    display: flex;
    justify-content: center;
    margin: 28px 0 44px;
  }

  .mascot-frame {
    position: relative;
    border-radius: 28px;
    padding: 10px;
    background: linear-gradient(135deg, var(--pop-pink), var(--primary), var(--pop-cyan));
    box-shadow: 0 12px 0 var(--border), 0 24px 48px rgba(0, 0, 0, 0.6);
    max-width: 800px;
    width: 100%;
    transition: transform 0.3s ease;
  }

  .mascot-frame:hover {
    transform: translateY(-4px) scale(1.01);
  }

  .mascot-img {
    width: 100%;
    height: auto;
    border-radius: 20px;
    display: block;
    object-fit: cover;
  }

  .mascot-badge {
    position: absolute;
    bottom: 24px;
    right: 24px;
    background: rgba(13, 11, 24, 0.88);
    backdrop-filter: blur(10px);
    border: 2px solid var(--pop-yellow);
    color: #fff;
    padding: 8px 16px;
    border-radius: 14px;
    font-family: 'JetBrains Mono', monospace;
    font-weight: 800;
    font-size: 0.9rem;
    box-shadow: 0 6px 16px rgba(0,0,0,0.6);
  }

  /* Clay Buttons */
  .btn-group {
    display: flex;
    justify-content: center;
    gap: 16px;
    flex-wrap: wrap;
    margin-bottom: 20px;
  }

  .clay-btn {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    background: var(--pop-pink);
    color: #ffffff;
    font-family: 'Space Grotesk', sans-serif;
    font-weight: 800;
    font-size: 1.05rem;
    padding: 14px 28px;
    border-radius: 16px;
    text-decoration: none;
    border: 2px solid #ff5ba2;
    box-shadow: 0 6px 0 #b3004b, 0 12px 20px rgba(255, 42, 133, 0.35);
    transition: all 0.15s ease;
    cursor: pointer;
  }

  .clay-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 0 #b3004b, 0 16px 25px rgba(255, 42, 133, 0.45);
  }

  .clay-btn:active {
    transform: translateY(4px);
    box-shadow: 0 2px 0 #b3004b;
  }

  .clay-btn-secondary {
    background: var(--surface-clay);
    color: var(--text-heading);
    border: 2px solid var(--border);
    box-shadow: 0 6px 0 var(--border), 0 12px 20px rgba(0,0,0,0.3);
  }

  .clay-btn-secondary:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 0 var(--border), 0 16px 25px rgba(0,0,0,0.4);
    border-color: var(--pop-cyan);
  }

  /* Terminal Window */
  .terminal-box {
    background: #090714;
    border: 2px solid var(--border);
    border-radius: 20px;
    box-shadow: 0 8px 0 var(--border), 0 20px 40px rgba(0,0,0,0.6);
    overflow: hidden;
    margin: 40px 0;
    text-align: left;
  }

  .terminal-header {
    background: #151128;
    padding: 12px 18px;
    display: flex;
    align-items: center;
    gap: 8px;
    border-bottom: 2px solid var(--border);
  }

  .dot { width: 12px; height: 12px; border-radius: 50%; display: inline-block; }
  .dot-red { background: #ef4444; }
  .dot-yellow { background: #f59e0b; }
  .dot-green { background: #10b981; }
  .terminal-title {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.85rem;
    color: var(--muted);
    margin-left: 8px;
    font-weight: 700;
  }

  .terminal-body {
    padding: 24px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.95rem;
    line-height: 1.7;
    color: #e2e8f0;
  }

  .prompt { color: var(--pop-pink); font-weight: 800; }
  .cmd { color: var(--pop-cyan); font-weight: 700; }
  .arg { color: var(--pop-yellow); }
  .out-success { color: var(--pop-green); }
  .out-comment { color: var(--muted); font-style: italic; margin-top: 12px; }

  /* TUI Agenda Box */
  .tui-agenda-box {
    background: rgba(34, 197, 94, 0.06);
    border: 1.5px solid #22c55e;
    border-radius: 14px;
    padding: 16px 20px;
    margin: 12px 0 16px;
    box-shadow: 0 4px 16px rgba(34, 197, 94, 0.12);
  }
  .tui-agenda-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 800;
    font-size: 0.95rem;
    color: #4ade80;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px dashed rgba(34, 197, 94, 0.3);
  }
  .tui-agenda-row {
    display: flex;
    align-items: center;
    gap: 14px;
    margin: 8px 0;
    font-size: 0.92rem;
  }
  .tui-agenda-time {
    color: var(--pop-yellow);
    font-weight: 700;
    background: rgba(250, 204, 21, 0.1);
    border: 1px solid rgba(250, 204, 21, 0.25);
    padding: 2px 10px;
    border-radius: 8px;
    font-size: 0.85rem;
  }
  .tui-agenda-text {
    color: #f8fafc;
    font-weight: 600;
  }

  /* Feature Clay Cards */
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 24px;
    margin: 48px 0;
  }

  .clay-card {
    background: var(--surface);
    border: 2px solid var(--border);
    border-radius: 20px;
    padding: 28px;
    box-shadow: 0 6px 0 var(--border), 0 12px 24px rgba(0,0,0,0.3);
    transition: transform 0.2s ease, border-color 0.2s ease;
  }

  .clay-card:hover {
    transform: translateY(-4px);
    border-color: var(--primary);
  }

  .card-icon {
    font-size: 2.2rem;
    margin-bottom: 16px;
    display: inline-block;
    background: var(--surface-clay);
    padding: 12px;
    border-radius: 16px;
    border: 2px solid var(--border);
    box-shadow: 0 4px 0 var(--border);
  }

  .clay-card h3 {
    font-size: 1.35rem;
    color: var(--text-heading);
    margin-bottom: 12px;
    font-weight: 800;
  }

  .clay-card p {
    color: var(--muted);
    font-size: 0.98rem;
  }

  /* Policy / Legal Pages */
  .doc-container {
    background: var(--surface);
    border: 2px solid var(--border);
    border-radius: 24px;
    padding: 40px;
    box-shadow: 0 8px 0 var(--border), 0 20px 40px rgba(0,0,0,0.4);
    margin: 32px 0;
  }

  .doc-container h1 { font-size: 2.4rem; margin-bottom: 12px; }
  .doc-container h2 { font-size: 1.5rem; color: var(--pop-cyan); margin: 32px 0 12px; font-weight: 800; }
  .doc-container p { margin-bottom: 16px; font-size: 1.05rem; }
  .doc-container ul { margin-left: 24px; margin-bottom: 20px; }
  .doc-container li { margin-bottom: 10px; font-size: 1rem; color: var(--text); }
  .doc-container code {
    background: var(--surface-clay);
    color: var(--pop-yellow);
    padding: 2px 8px;
    border-radius: 6px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.9rem;
    border: 1px solid var(--border);
  }

  footer {
    margin-top: 64px;
    padding-top: 28px;
    border-top: 2px solid var(--border);
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.9rem;
    color: var(--muted);
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 16px;
  }

  footer a { color: var(--text); text-decoration: none; font-weight: 700; transition: color 0.2s; }
  footer a:hover { color: var(--pop-pink); }
`;

function renderLayout(title: string, content: string): string {
  return `<!DOCTYPE html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${title} — Mỹ Lích (Lich)</title>
  <link rel="icon" href="data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>🗓️</text></svg>">
  <style>${popStyles}</style>
</head>
<body>
  <div class="container">
    <header>
      <a href="/" class="logo">
        <span class="logo-badge">MỸ LÍCH</span>
        <span>lich</span>
      </a>
      <nav>
        <a href="/">Trang chủ</a>
        <a href="/privacy">Bảo mật</a>
        <a href="/terms">Điều khoản</a>
        <a href="https://github.com/spiderdev-vn/mylich" target="_blank" rel="noopener">★ GitHub</a>
      </nav>
    </header>
    <main>
      ${content}
    </main>
    <footer>
      <div>© 2026 <strong>Mỹ Lích (Lich)</strong>. Lịch của mình, mình tính! 🚀</div>
      <div>
        <a href="/privacy">Privacy Policy</a> · 
        <a href="/terms">Terms of Service</a> · 
        <a href="https://github.com/spiderdev-vn/mylich" target="_blank" rel="noopener">GitHub</a>
      </div>
    </footer>
  </div>
</body>
</html>`;
}

function tryReadImage(filename: string): Buffer | null {
  const candidates = [
    path.resolve(process.cwd(), 'public', filename),
    path.resolve(process.cwd(), 'lich-server', 'public', filename),
    path.resolve(process.cwd(), '..', 'public', filename),
  ];

  for (const p of candidates) {
    if (fs.existsSync(p)) {
      try {
        return fs.readFileSync(p);
      } catch {
        // continue
      }
    }
  }
  return null;
}

export const publicRoutes: FastifyPluginAsync = async (app) => {
  // Static image routes
  app.get('/mascot.jpg', async (_request, reply) => {
    const buf = tryReadImage('mascot.jpg');
    if (buf) {
      reply.header('Cache-Control', 'public, max-age=86400');
      reply.type('image/jpeg');
      return buf;
    }
    return reply.status(404).send('Image not found');
  });

  app.get('/hero.jpg', async (_request, reply) => {
    const buf = tryReadImage('hero.jpg');
    if (buf) {
      reply.header('Cache-Control', 'public, max-age=86400');
      reply.type('image/jpeg');
      return buf;
    }
    return reply.status(404).send('Image not found');
  });

  // 1. Home / Landing Page (Fun Pop Claymorphic Terminal)
  app.get('/', async (_request, reply) => {
    reply.type('text/html; charset=utf-8');
    const content = `
      <section class="hero">
        <div class="hero-tag">✨ POP & FUN TERMINAL CALENDAR ✨</div>
        <h1>Lịch của mình, <span class="gradient-text">mình tính.</span></h1>
        <p class="tagline">Một cái lịch vui vẻ, siêu tốc, local-first cho Terminal (CLI/TUI) và tự host backend riêng tư!</p>
        
        <div class="btn-group">
          <a href="https://github.com/spiderdev-vn/mylich/releases" class="clay-btn" target="_blank" rel="noopener">
            <span>⚡ Cài đặt CLI Ngay</span>
          </a>
          <a href="https://github.com/spiderdev-vn/mylich" class="clay-btn clay-btn-secondary" target="_blank" rel="noopener">
            <span>📂 Xem Mã Nguồn GitHub</span>
          </a>
        </div>

        <!-- 3D Clay Mascot Hero Showcase -->
        <div class="mascot-container">
          <div class="mascot-frame">
            <img src="/mascot.jpg" alt="Mỹ Lích 3D Clay Mascot" class="mascot-img">
            <div class="mascot-badge">Lich v0.2.0 • Local-First 💻</div>
          </div>
        </div>
      </section>

      <!-- Terminal Preview Mockup -->
      <div class="terminal-box">
        <div class="terminal-header">
          <span class="dot dot-red"></span>
          <span class="dot dot-yellow"></span>
          <span class="dot dot-green"></span>
          <span class="terminal-title">kunix@pop-terminal: ~ (lich-cli)</span>
        </div>
        <div class="terminal-body">
          <div class="out-comment"># 1. Thêm sự kiện tức thì (phản hồi trong 1 mili-giây)</div>
          <div><span class="prompt">$</span> <span class="cmd">lich add</span> <span class="arg">"Ăn tối với người yêu"</span> --at <span class="arg">19:00</span></div>
          <div class="out-success">✓ Đã tạo sự kiện: Ăn tối với người yêu [19:00 - 20:00 20/08/2026]</div>

          <div class="out-comment"># 2. Xem lịch trình hôm nay dạng TUI / Agenda siêu đẹp</div>
          <div><span class="prompt">$</span> <span class="cmd">lich today</span></div>
          <div class="tui-agenda-box">
            <div class="tui-agenda-header">🗓️ LỊCH HÔM NAY (MỸ LÍCH)</div>
            <div class="tui-agenda-row">
              <span class="tui-agenda-time">10:00 - 11:30</span>
              <span class="tui-agenda-text">Họp kiến trúc hệ thống</span>
            </div>
            <div class="tui-agenda-row">
              <span class="tui-agenda-time">19:00 - 20:00</span>
              <span class="tui-agenda-text">Ăn tối với người yêu 💕</span>
            </div>
          </div>

          <div class="out-comment"># 3. Đồng bộ 2 chiều với Google Calendar (Zero-Config)</div>
          <div><span class="prompt">$</span> <span class="cmd">lich google sync</span></div>
          <div class="out-success">✓ Đồng bộ 2 chiều thành công: 1 đẩy lên Google, 0 kéo về.</div>
        </div>
      </div>

      <!-- Feature Clay Cards -->
      <div class="grid">
        <div class="clay-card">
          <div class="card-icon">⚡</div>
          <h3>Local-first Siêu Tốc</h3>
          <p>Mỹ Lích lưu trữ toàn bộ dữ liệu trong SQLite máy bạn. Mất mạng vẫn tạo/sửa lịch vèo vèo, có mạng thì tự động đồng bộ ngầm!</p>
        </div>
        <div class="clay-card">
          <div class="card-icon">🔄</div>
          <h3>Google Sync 2 Chiều</h3>
          <p>Đồng bộ thông minh với Google Calendar. Áp dụng Last-Write-Wins giúp mọi thay đổi ở điện thoại hay terminal luôn chuẩn xác.</p>
        </div>
        <div class="clay-card">
          <div class="card-icon">🛡️</div>
          <h3>Self-hosted & Riêng Tư</h3>
          <p>Dữ liệu lịch là của bạn, không phải của các cỗ máy quảng cáo. Tự host backend riêng qua Docker chỉ với 1 lệnh compose!</p>
        </div>
      </div>
    `;
    return renderLayout('Trang chủ', content);
  });

  // 2. Privacy Policy (Đáp ứng chuẩn Google OAuth Verification)
  app.get('/privacy', async (_request, reply) => {
    reply.type('text/html; charset=utf-8');
    const content = `
      <div class="doc-container">
        <h1>🔒 Chính Sách Bảo Mật (Privacy Policy)</h1>
        <p style="color: var(--pop-yellow); font-family: 'JetBrains Mono', monospace; font-size: 0.95rem;">Cập nhật lần cuối: Ngày 19 tháng 08 năm 2026</p>
        <br>
        <p><strong>Mỹ Lích (Lich)</strong> cam kết tôn trọng và bảo vệ quyền riêng tư tuyệt đối của bạn. Dữ liệu lịch trình là tài sản cá nhân nhạy cảm và quan trọng nhất của mỗi người dùng.</p>

        <h2>1. Thông tin chúng tôi truy cập</h2>
        <p>Khi bạn kích hoạt tính năng tích hợp Google Calendar, ứng dụng sẽ yêu cầu các quyền truy cập được cấp phép bởi bạn:</p>
        <ul>
          <li><strong>Địa chỉ Email (Google User Profile)</strong>: Dùng để xác thực và hiển thị tài khoản nào đang được liên kết trên máy chủ.</li>
          <li><strong>Dữ liệu Lịch (Google Calendar API Scope <code>https://www.googleapis.com/auth/calendar</code>)</strong>: Đọc và ghi danh sách lịch và các sự kiện lịch (tiêu đề, thời gian, mô tả, địa điểm) để phục vụ đồng bộ 2 chiều giữa Mỹ Lích và Google Calendar của bạn.</li>
        </ul>

        <h2>2. Mục đích sử dụng dữ liệu</h2>
        <p>Chúng tôi <strong>CHỈ</strong> sử dụng dữ liệu từ Google Calendar cho các mục đích thiết yếu:</p>
        <ul>
          <li>Hiển thị lịch trình và sự kiện của bạn trên giao diện CLI/TUI của Mỹ Lích.</li>
          <li>Đồng bộ hóa 2 chiều: Đẩy sự kiện từ Mỹ Lích lên Google và kéo sự kiện mới từ Google về Mỹ Lích.</li>
        </ul>

        <h2>3. Lưu trữ và Bảo mật Dữ liệu</h2>
        <ul>
          <li><strong>Máy chủ tự quản lý (Self-hosted)</strong>: Dữ liệu lịch và Refresh Token được lưu trữ trên cơ sở dữ liệu SQLite riêng của máy chủ do chính bạn hoặc quản trị viên hệ thống của bạn triển khai.</li>
          <li><strong>Tuyệt đối không chia sẻ với bên thứ ba</strong>: Chúng tôi không bao giờ bán, cho thuê, chia sẻ hay chuyển giao dữ liệu lịch của bạn cho bất kỳ bên thứ ba, nhà quảng cáo, mạng lưới tiếp thị hoặc mô hình huấn luyện AI nào.</li>
          <li><strong>Mã hóa bảo mật</strong>: Tất cả các kết nối trao đổi dữ liệu với Google API đều được mã hóa an toàn qua giao thức HTTPS/TLS.</li>
        </ul>

        <h2>4. Tuân thủ Chính sách Dữ liệu Người dùng của Google</h2>
        <p>Mỹ Lích hoàn toàn tuân thủ <a href="https://developers.google.com/terms/api-services-user-data-policy" target="_blank" rel="noopener" style="color: var(--pop-cyan); font-weight: 700;">Google API Services User Data Policy</a>, bao gồm các yêu cầu về <em>Limited Use (Sử dụng có giới hạn)</em>.</p>

        <h2>5. Quyền kiểm soát và Xóa dữ liệu</h2>
        <ul>
          <li><strong>Hủy liên kết tài khoản Google</strong>: Bạn có thể thu hồi quyền truy cập của Mỹ Lích bất kỳ lúc nào bằng lệnh CLI: <code>lich google disconnect</code> hoặc tại trang <a href="https://myaccount.google.com/permissions" target="_blank" rel="noopener" style="color: var(--pop-cyan); font-weight: 700;">Google Account Permissions</a>.</li>
          <li><strong>Xóa sạch toàn bộ dữ liệu</strong>: Bạn có thể xóa sạch dữ liệu trên máy chủ bằng lệnh: <code>lich nuke-database --remote</code>.</li>
        </ul>

        <h2>6. Liên hệ</h2>
        <p>Mọi thắc mắc về Chính sách bảo mật, vui lòng mở Issue tại kho mã nguồn chính thức: <a href="https://github.com/spiderdev-vn/mylich" target="_blank" rel="noopener" style="color: var(--pop-pink); font-weight: 700;">https://github.com/spiderdev-vn/mylich</a>.</p>
      </div>
    `;
    return renderLayout('Chính sách bảo mật', content);
  });

  // 3. Terms of Service
  app.get('/terms', async (_request, reply) => {
    reply.type('text/html; charset=utf-8');
    const content = `
      <div class="doc-container">
        <h1>📜 Điều Khoản Dịch Vụ (Terms of Service)</h1>
        <p style="color: var(--pop-yellow); font-family: 'JetBrains Mono', monospace; font-size: 0.95rem;">Cập nhật lần cuối: Ngày 19 tháng 08 năm 2026</p>
        <br>
        <h2>1. Giới thiệu</h2>
        <p>Chào mừng bạn đến với <strong>Mỹ Lích (Lich)</strong> — giải pháp quản lý lịch cá nhân local-first mã nguồn mở. Khi sử dụng phần mềm hoặc dịch vụ máy chủ của chúng tôi, bạn đồng ý tuân thủ các điều khoản sau.</p>

        <h2>2. Giấy phép & Mã nguồn mở</h2>
        <p>Mỹ Lích được phát hành theo giấy phép <strong>MIT License</strong>. Bạn có quyền tự do sử dụng, chỉnh sửa, phân phối và triển khai tự host phục vụ mục đích cá nhân hoặc thương mại.</p>

        <h2>3. Trách nhiệm người dùng</h2>
        <ul>
          <li>Bạn chịu trách nhiệm bảo mật thông tin đăng nhập, JWT Secret và dữ liệu lưu trữ trên máy chủ tự host của mình.</li>
          <li>Không sử dụng dịch vụ vào các mục đích vi phạm pháp luật hoặc can thiệp trái phép vào hệ thống của bên thứ ba.</li>
        </ul>

        <h2>4. Dịch vụ Tích hợp Bên thứ ba</h2>
        <p>Khi bạn kích hoạt đồng bộ hóa Google Calendar, bạn đồng thời chịu sự ràng buộc bởi Điều khoản dịch vụ của Google.</p>

        <h2>5. Từ chối bảo đảm (Disclaimer)</h2>
        <p>Phần mềm được cung cấp "nguyên trạng" (AS IS), không có bảo đảm đi kèm. Các tác giả không chịu trách nhiệm đối với bất kỳ mất mát dữ liệu hoặc thiệt hại phát sinh từ việc sử dụng phần mềm.</p>
      </div>
    `;
    return renderLayout('Điều khoản dịch vụ', content);
  });
};
