const bcrypt = require('bcryptjs');
const mysql = require('mysql2/promise');

(async () => {
  const dbHost = process.env.DB_HOST || 'db';
  const dbUser = process.env.DB_USER || 'salfanet_user';
  const dbPassword = process.env.DB_PASSWORD;
  const dbName = process.env.DB_NAME || 'salfanet_radius';

  if (!dbPassword) {
    console.error('DB_PASSWORD is required');
    process.exit(1);
  }

  const conn = await mysql.createConnection({
    host: dbHost,
    user: dbUser,
    password: dbPassword,
    database: dbName,
  });

  const [rows] = await conn.execute('SELECT COUNT(*) as count FROM admin_users WHERE role = ?', ['SUPER_ADMIN']);
  if (rows[0].count > 0) {
    console.log('Super admin already exists, skipping seed.');
    await conn.end();
    return;
  }

  const hash = bcrypt.hashSync('admin123', 10);
  const now = new Date();
  await conn.execute(
    'INSERT INTO admin_users (id,username,email,password,name,role,isActive,twoFactorEnabled,createdAt,updatedAt) VALUES (?,?,?,?,?,?,?,?,?,?)',
    ['admin-superadmin', 'superadmin', 'admin@example.com', hash, 'Super Administrator', 'SUPER_ADMIN', 1, 0, now, now]
  );
  console.log('Super admin seeded: superadmin / admin123');
  await conn.end();
})().catch(err => {
  console.error('Seed error:', err.message);
  process.exit(1);
});
