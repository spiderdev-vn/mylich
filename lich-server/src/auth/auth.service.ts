import crypto from 'node:crypto';
import * as jose from 'jose';
import type { UserRepository, User } from '../db/repositories/user.repository.ts';
import type { CalendarRepository } from '../db/repositories/calendar.repository.ts';
import { BadRequestError, UnauthorizedError, ConflictError } from '../common/errors.ts';

export class AuthService {
  private userRepo: UserRepository;
  private calendarRepo: CalendarRepository;
  private secretKey: Uint8Array;

  constructor(
    userRepo: UserRepository,
    calendarRepo: CalendarRepository,
    jwtSecret: string
  ) {
    this.userRepo = userRepo;
    this.calendarRepo = calendarRepo;
    this.secretKey = new TextEncoder().encode(jwtSecret);
  }

  public hashPassword(password: string): string {
    const salt = crypto.randomBytes(16).toString('hex');
    const hash = crypto.scryptSync(password, salt, 64).toString('hex');
    return `${salt}:${hash}`;
  }

  public verifyPassword(password: string, combined: string): boolean {
    const [salt, storedHash] = combined.split(':');
    if (!salt || !storedHash) {
      return false;
    }
    const computedHash = crypto.scryptSync(password, salt, 64).toString('hex');
    try {
      return crypto.timingSafeEqual(
        Buffer.from(storedHash, 'hex'),
        Buffer.from(computedHash, 'hex')
      );
    } catch {
      return false;
    }
  }

  public async generateToken(user: User): Promise<string> {
    return new jose.SignJWT({ userId: user.id, username: user.username })
      .setProtectedHeader({ alg: 'HS256' })
      .setIssuedAt()
      .setExpirationTime('30d')
      .sign(this.secretKey);
  }

  public async verifyToken(token: string): Promise<{ userId: string; username: string }> {
    try {
      const { payload } = await jose.jwtVerify(token, this.secretKey);
      return {
        userId: payload.userId as string,
        username: payload.username as string,
      };
    } catch {
      throw new UnauthorizedError('Invalid or expired authentication token');
    }
  }

  public async register(username: string, password: string, timezone = 'UTC'): Promise<{ token: string; user: { id: string; username: string } }> {
    if (!username || typeof username !== 'string' || username.trim().length < 2) {
      throw new BadRequestError('Username must be at least 2 characters long');
    }
    if (!password || typeof password !== 'string' || password.length < 4) {
      throw new BadRequestError('Password must be at least 4 characters long');
    }

    const trimmedUsername = username.trim();
    const existing = this.userRepo.findByUsername(trimmedUsername);
    if (existing) {
      throw new ConflictError(`User with username '${trimmedUsername}' already exists`);
    }

    const userId = crypto.randomUUID();
    const passwordHash = this.hashPassword(password);
    const user = this.userRepo.create({
      id: userId,
      username: trimmedUsername,
      password_hash: passwordHash,
    });

    // Auto-create default "Personal" calendar
    const calendarId = crypto.randomUUID();
    this.calendarRepo.create({
      id: calendarId,
      user_id: user.id,
      name: 'Personal',
      description: 'Default personal calendar',
      timezone,
      is_default: true,
    });

    const token = await this.generateToken(user);
    return {
      token,
      user: {
        id: user.id,
        username: user.username,
      },
    };
  }

  public async login(username: string, password: string): Promise<{ token: string; user: { id: string; username: string } }> {
    if (!username || !password) {
      throw new BadRequestError('Username and password are required');
    }

    const user = this.userRepo.findByUsername(username.trim());
    if (!user) {
      throw new UnauthorizedError('Invalid username or password');
    }

    const isValid = this.verifyPassword(password, user.password_hash);
    if (!isValid) {
      throw new UnauthorizedError('Invalid username or password');
    }

    const token = await this.generateToken(user);
    return {
      token,
      user: {
        id: user.id,
        username: user.username,
      },
    };
  }
}
