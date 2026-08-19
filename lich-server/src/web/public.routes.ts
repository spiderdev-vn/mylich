import type { FastifyPluginAsync } from 'fastify';

const baseStyles = `
  :root {
    --bg: #0d1117;
    --surface: #161b22;
    --border: #30363d;
    --primary: #58a6ff;
    --primary-hover: #79c0ff;
    --text: #c9d1d9;
    --text-heading: #f0f6fc;
    --muted: #8b949e;
    --success: #3fb950;
    --accent-purple: #bc8cff;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
    background-color: var(--bg);
    color: var(--text);
    line-height: 1.6;
    padding: 0;
  }
  .container {
    max-width: 860px;
    margin: 0 auto;
    padding: 40px 20px 80px;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: 24px;
    border-bottom: 1px solid var(--border);
    margin-bottom: 40px;
  }
  .logo {
    display: flex;
    align-items: center;
    gap: 12px;
    font-weight: 700;
    font-size: 1.3rem;
    color: var(--text-heading);
    text-decoration: none;
  }
  .logo-badge {
    background: linear-gradient(135deg, #6e40c9, #1f6feb);
    color: #fff;
    padding: 4px 10px;
    border-radius: 8px;
    font-size: 0.9rem;
    letter-spacing: 0.5px;
  }
  nav a {
    color: var(--muted);
    text-decoration: none;
    margin-left: 20px;
    font-size: 0.95rem;
    transition: color 0.2s;
  }
  nav a:hover { color: var(--primary); }
  h1 { font-size: 2.4rem; font-weight: 800; color: var(--text-heading); margin-bottom: 16px; letter-spacing: -0.5px; }
  h2 { font-size: 1.5rem; font-weight: 700; color: var(--text-heading); margin-top: 36px; margin-bottom: 12px; }
  h3 { font-size: 1.2rem; font-weight: 600; color: var(--text-heading); margin-top: 24px; margin-bottom: 8px; }
  p { margin-bottom: 16px; color: var(--text); }
  ul, ol { margin-left: 24px; margin-bottom: 20px; }
  li { margin-bottom: 8px; }
  .tagline {
    font-size: 1.25rem;
    color: var(--muted);
    margin-bottom: 32px;
  }
  .hero-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 32px;
    margin-bottom: 40px;
    box-shadow: 0 8px 24px rgba(0,0,0,0.4);
  }
  .code-block {
    background: #090d13;
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 16px 20px;
    font-family: 'JetBrains Mono', Consolas, Monaco, monospace;
    font-size: 0.95rem;
    color: #58a6ff;
    overflow-x: auto;
    margin: 16px 0 24px;
  }
  .btn {
    display: inline-block;
    background: #238636;
    color: #ffffff;
    padding: 10px 20px;
    border-radius: 6px;
    font-weight: 600;
    text-decoration: none;
    transition: background 0.2s;
  }
  .btn:hover { background: #2ea043; }
  .btn-outline {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-heading);
    margin-left: 12px;
  }
  .btn-outline:hover { background: var(--surface); border-color: var(--muted); }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 20px;
    margin: 28px 0;
  }
  .card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 24px;
  }
  .card h3 { margin-top: 0; color: var(--primary); }
  footer {
    margin-top: 60px;
    padding-top: 24px;
    border-top: 1px solid var(--border);
    font-size: 0.9rem;
    color: var(--muted);
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 16px;
  }
  footer a { color: var(--muted); text-decoration: none; }
  footer a:hover { color: var(--primary); }
`;

function renderLayout(title: string, content: string): string {
  return `<!DOCTYPE html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${title} — Mỹ Lích</title>
  <style>${baseStyles}</style>
</head>
<body>
  <div class="container">
    <header>
      <a href="/" class="logo">
        <span class="logo-badge">MỸ LÍCH</span>
        <span>Lich</span>
      </a>
      <nav>
        <a href="/">Trang chủ</a>
        <a href="/privacy">Chính sách bảo mật</a>
        <a href="/terms">Điều khoản dịch vụ</a>
        <a href="https://github.com/spiderdev-vn/mylich" target="_blank" rel="noopener">GitHub</a>
      </nav>
    </header>
    <main>
      ${content}
    </main>
    <footer>
      <div>© 2026 Mỹ Lích (Lich). All rights reserved.</div>
      <div>
        <a href="/privacy">Privacy Policy</a> · 
        <a href="/terms">Terms of Service</a> · 
        <a href="https://github.com/spiderdev-vn/mylich" target="_blank" rel="noopener">GitHub Repository</a>
      </div>
    </footer>
  </div>
</body>
</html>`;
}

export const publicRoutes: FastifyPluginAsync = async (app) => {
  // 1. Home / Landing Page
  app.get('/', async (_request, reply) => {
    reply.type('text/html; charset=utf-8');
    const content = `
      <h1>Mỹ Lích — Lịch của mình, mình tính</h1>
      <p class="tagline">Hệ thống lịch cá nhân Local-first cho Terminal (CLI/TUI) và tự host backend riêng tư.</p>

      <div class="hero-card">
        <h2>Bắt đầu sử dụng chỉ với một lệnh</h2>
        <p>Thêm sự kiện, kiểm tra lịch trình hôm nay, và đồng bộ tự động 2 chiều với Google Calendar tức thì.</p>
        <div class="code-block">
$ lich add "Họp dự án lúc 10h sáng" --at 10:00<br>
$ lich google sync
        </div>
        <div>
          <a href="https://github.com/spiderdev-vn/mylich/releases" class="btn" target="_blank" rel="noopener">Tải xuống CLI Binary</a>
          <a href="https://github.com/spiderdev-vn/mylich" class="btn btn-outline" target="_blank" rel="noopener">Xem mã nguồn GitHub</a>
        </div>
      </div>

      <h2>Tính năng nổi bật</h2>
      <div class="grid">
        <div class="card">
          <h3>⚡ Local-first & Siêu Nhanh</h3>
          <p>Dữ liệu lưu trữ trong SQLite cục bộ. Mọi thao tác thêm, sửa, xóa sự kiện phản hồi tức thì mà không cần chờ mạng.</p>
        </div>
        <div class="card">
          <h3>🔄 Đồng bộ Google Calendar 2 chiều</h3>
          <p>Tự động so khớp lịch với Google Calendar theo cơ chế Last-Write-Wins, giữ lịch trình của bạn luôn đồng nhất ở mọi nơi.</p>
        </div>
        <div class="card">
          <h3>🔒 Tự chủ & Riêng tư (Self-hosted)</h3>
          <p>Mỹ Lích thuộc về bạn. Tự host backend riêng bằng Docker, dữ liệu hoàn toàn do bạn quản lý và kiểm soát.</p>
        </div>
      </div>
    `;
    return renderLayout('Trang chủ', content);
  });

  // 2. Privacy Policy
  app.get('/privacy', async (_request, reply) => {
    reply.type('text/html; charset=utf-8');
    const content = `
      <h1>Chính Sách Bảo Mật (Privacy Policy)</h1>
      <p class="tagline">Cập nhật lần cuối: Ngày 19 tháng 08 năm 2026</p>

      <div class="hero-card">
        <p><strong>Mỹ Lích (Lich)</strong> cam kết tôn trọng và bảo vệ quyền riêng tư của bạn. Chúng tôi coi dữ liệu lịch là thông tin cá nhân nhạy cảm và quan trọng nhất của người dùng.</p>
      </div>

      <h2>1. Thông tin chúng tôi thu thập</h2>
      <p>Khi bạn sử dụng Mỹ Lích và kích hoạt tính năng tích hợp Google Calendar, ứng dụng sẽ yêu cầu các quyền truy cập sau:</p>
      <ul>
        <li><strong>Địa chỉ Email (Google User Profile)</strong>: Dùng để xác thực và hiển thị tài khoản Google nào đang được liên kết.</li>
        <li><strong>Dữ liệu Lịch (Google Calendar API Scope <code>https://www.googleapis.com/auth/calendar</code>)</strong>: Đọc và ghi danh sách lịch cũng như các sự kiện lịch (tiêu đề, thời gian, địa điểm, mô tả) nhằm phục vụ chức năng đồng bộ 2 chiều giữa Mỹ Lích và Google Calendar của bạn.</li>
      </ul>

      <h2>2. Mục đích sử dụng dữ liệu</h2>
      <p>Chúng tôi <strong>CHỈ</strong> sử dụng dữ liệu được cấp quyền từ Google Calendar cho các mục đích sau:</p>
      <ul>
        <li>Hiển thị lịch trình và sự kiện của bạn trên giao diện CLI/TUI của Mỹ Lích.</li>
        <li>Đồng bộ hóa 2 chiều: Khi bạn tạo/sửa sự kiện trên Mỹ Lích, sự kiện được cập nhật lên Google Calendar; và ngược lại khi có sự kiện mới trên Google Calendar, sự kiện được kéo về ứng dụng Mỹ Lích của bạn.</li>
      </ul>

      <h2>3. Lưu trữ và Bảo mật Dữ liệu</h2>
      <ul>
        <li><strong>Máy chủ tự quản lý (Self-hosted)</strong>: Toàn bộ dữ liệu lịch và Refresh Token được lưu trữ trên cơ sở dữ liệu SQLite riêng của máy chủ do chính bạn hoặc quản trị viên hệ thống của bạn triển khai.</li>
        <li><strong>Không chia sẻ với bên thứ ba</strong>: Chúng tôi không bao giờ bán, cho thuê, chia sẻ hay chuyển giao dữ liệu lịch của bạn cho bất kỳ bên thứ ba, nhà quảng cáo, mạng lưới tiếp thị hoặc mô hình huấn luyện AI nào.</li>
        <li><strong>Mã hóa bảo mật</strong>: Tất cả các kết nối trao đổi dữ liệu với Google Calendar API đều được thực hiện qua giao thức mã hóa HTTPS/TLS an toàn.</li>
      </ul>

      <h2>4. Tuân thủ Chính sách Dữ liệu Người dùng của Google</h2>
      <p>Mỹ Lích hoàn toàn tuân thủ <a href="https://developers.google.com/terms/api-services-user-data-policy" target="_blank" rel="noopener" style="color: var(--primary);">Google API Services User Data Policy</a>, bao gồm các yêu cầu về <em>Limited Use (Sử dụng có giới hạn)</em>.</p>

      <h2>5. Quyền kiểm soát và Xóa dữ liệu của bạn</h2>
      <ul>
        <li><strong>Hủy liên kết tài khoản Google</strong>: Bạn có thể thu hồi quyền truy cập của Mỹ Lích bất kỳ lúc nào bằng lệnh CLI: <code>lich google disconnect</code> hoặc trực tiếp tại trang quản lý tài khoản Google: <a href="https://myaccount.google.com/permissions" target="_blank" rel="noopener" style="color: var(--primary);">Google Account Permissions</a>.</li>
        <li><strong>Xóa toàn bộ dữ liệu</strong>: Bạn có thể xóa sạch toàn bộ dữ liệu trên máy chủ bằng lệnh: <code>lich nuke-database --remote</code>.</li>
      </ul>

      <h2>6. Liên hệ</h2>
      <p>Nếu bạn có bất kỳ câu hỏi nào về Chính sách bảo mật này, vui lòng liên hệ hoặc mở Issue tại kho mã nguồn chính thức: <a href="https://github.com/spiderdev-vn/mylich" target="_blank" rel="noopener" style="color: var(--primary);">https://github.com/spiderdev-vn/mylich</a>.</p>
    `;
    return renderLayout('Chính sách bảo mật', content);
  });

  // 3. Terms of Service
  app.get('/terms', async (_request, reply) => {
    reply.type('text/html; charset=utf-8');
    const content = `
      <h1>Điều Khoản Dịch Vụ (Terms of Service)</h1>
      <p class="tagline">Cập nhật lần cuối: Ngày 19 tháng 08 năm 2026</p>

      <h2>1. Giới thiệu</h2>
      <p>Chào mừng bạn đến với <strong>Mỹ Lích (Lich)</strong> — giải pháp quản lý lịch cá nhân local-first mã nguồn mở. Bằng cách sử dụng phần mềm hoặc dịch vụ máy chủ của chúng tôi, bạn đồng ý tuân thủ các điều khoản sau.</p>

      <h2>2. Giấy phép & Sử dụng mã nguồn</h2>
      <p>Mỹ Lích là phần mềm mã nguồn mở được phát hành theo giấy phép MIT License. Bạn có quyền tự do sử dụng, chỉnh sửa, phân phối và triển khai tự host phục vụ mục đích cá nhân hoặc thương mại theo các điều khoản của giấy phép MIT.</p>

      <h2>3. Trách nhiệm của người dùng</h2>
      <ul>
        <li>Bạn chịu trách nhiệm bảo mật thông tin đăng nhập, JWT Secret và dữ liệu lưu trữ trên máy chủ tự host của mình.</li>
        <li>Không sử dụng dịch vụ để thực hiện các hành vi vi phạm pháp luật hoặc can thiệp trái phép vào hệ thống của bên thứ ba (bao gồm dịch vụ của Google).</li>
      </ul>

      <h2>4. Dịch vụ Tích hợp Bên thứ ba (Google Calendar)</h2>
      <p>Khi bạn kích hoạt tính năng đồng bộ hóa với Google Calendar, bạn đồng thời chịu sự ràng buộc bởi các Điều khoản dịch vụ và Chính sách của Google. Mỹ Lích không chịu trách nhiệm đối với sự gián đoạn dịch vụ xuất phát từ phía nhà cung cấp bên thứ ba.</p>

      <h2>5. Từ chối bảo đảm (Disclaimer)</h2>
      <p>Phần mềm được cung cấp "nguyên trạng" (AS IS), không có bất kỳ sự bảo đảm rõ ràng hay ngụ ý nào. Các tác giả không chịu trách nhiệm đối với bất kỳ mất mát dữ liệu hoặc thiệt hại phát sinh từ việc sử dụng phần mềm.</p>

      <h2>6. Thay đổi điều khoản</h2>
      <p>Chúng tôi có thể cập nhật các điều khoản này khi cần thiết. Mọi thay đổi sẽ được công bố trực tiếp tại trang web này và trên kho mã nguồn GitHub.</p>
    `;
    return renderLayout('Điều khoản dịch vụ', content);
  });
};
