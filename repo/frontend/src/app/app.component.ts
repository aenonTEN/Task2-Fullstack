import { Component, OnInit } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { RouterModule, Router } from "@angular/router";
import { ApiService } from "./api.service";

@Component({
  selector: "app-root",
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule],
  template: `
    <main class="container">
      <header class="header">
        <div class="header-content">
          <h1>Integrated Platform</h1>
          <span class="app-subtitle">Enterprise Recruitment Management</span>
        </div>
        <div class="meta">
          <span class="pill" [class.ok]="status === 'authenticated'" [class.bad]="status === 'error'">
            {{ status }}
          </span>
        </div>
      </header>

      <section *ngIf="status !== 'authenticated'" class="login-section">
        <div class="glass-panel login-panel">
          <div class="login-header">
            <h2>Sign In</h2>
            <p>Enter your credentials to access the platform</p>
          </div>
          <div class="grid">
            <label>
              <span>Username</span>
              <input [(ngModel)]="username" autocomplete="username" placeholder="Enter your username" />
            </label>
            <label>
              <span>Password</span>
              <input [(ngModel)]="password" type="password" autocomplete="current-password" placeholder="Enter your password" />
            </label>
          </div>
          <div *ngIf="message" class="error-message">{{ message }}</div>
          <div class="row">
            <button (click)="login()" [disabled]="busy" class="login-btn">
              {{ busy ? 'Signing in...' : 'Sign In' }}
            </button>
          </div>
          <p class="login-hint">Contact your administrator if you don't have credentials</p>
        </div>
      </section>

      <section *ngIf="status === 'authenticated'" class="glass-panel">
        <div class="session-header">
          <h2>Session</h2>
          <button class="btn-secondary btn-small" (click)="logout()" [disabled]="busy">Sign Out</button>
        </div>
        <div class="session-info">
          <div class="info-item">
            <span class="info-label">Expires</span>
            <span class="info-value">{{ expiresAt | date:'medium' }}</span>
          </div>
          <div *ngIf="api.me" class="info-item">
            <span class="info-label">User</span>
            <span class="info-value">{{ api.me.userId }}</span>
          </div>
          <div *ngIf="api.me" class="info-item">
            <span class="info-label">Roles</span>
            <div class="tag-list">
              <span *ngFor="let role of api.me.roleIds" class="tag">{{ role }}</span>
            </div>
          </div>
        </div>
      </section>

      <nav *ngIf="status === 'authenticated'" class="nav-pills">
        <button routerLink="/recruitment" routerLinkActive="active">Recruitment</button>
        <button routerLink="/compliance" routerLinkActive="active">Compliance</button>
        <button routerLink="/cases" routerLinkActive="active">Cases</button>
        <button routerLink="/audit" routerLinkActive="active">Audit</button>
        <button routerLink="/tags" routerLinkActive="active">Tags</button>
      </nav>

      <router-outlet *ngIf="status === 'authenticated'"></router-outlet>

      <div *ngIf="showConfirmDialog" class="confirm-overlay" (click)="cancelConfirm()">
        <div class="confirm-dialog" (click)="$event.stopPropagation()">
          <h3>{{ confirmTitle }}</h3>
          <p>{{ confirmMessage }}</p>
          <div class="confirm-actions">
            <button class="btn-secondary" (click)="cancelConfirm()">Cancel</button>
            <button class="btn-danger" (click)="confirmAction()">Confirm</button>
          </div>
        </div>
      </div>
    </main>
  `
})
export class AppComponent implements OnInit {
  username = "";
  password = "";

  status: "anonymous" | "authenticated" | "error" = "anonymous";
  busy = false;
  message = "";
  expiresAt = "";

  showConfirmDialog = false;
  confirmTitle = "";
  confirmMessage = "";
  confirmCallback: (() => void) | null = null;

  constructor(public api: ApiService, private router: Router) { }

  ngOnInit() {
    if (this.api.token) {
      this.status = "authenticated";
      this.loadMe();
    }
  }

  async login() {
    this.busy = true;
    this.message = "";
    this.status = "anonymous";
    try {
      const data = await this.api.login(this.username, this.password);
      this.api.token = data.accessToken ?? "";
      this.expiresAt = data.expiresAt ?? "";
      this.status = "authenticated";
      localStorage.setItem("accessToken", this.api.token);
      await this.loadMe();
    } catch (e: any) {
      this.status = "error";
      this.message = e.error?.message || e.message || "Login failed";
    } finally {
      this.busy = false;
    }
  }

  async loadMe() {
    try {
      this.api.me = await this.api.getMe();
    } catch {
      this.api.me = null;
    }
  }

  async logout() {
    this.busy = true;
    this.message = "";
    try {
      await this.api.logout();
    } catch {
      // ignore
    } finally {
      localStorage.removeItem("accessToken");
      this.api.token = "";
      this.expiresAt = "";
      this.api.me = null;
      this.status = "anonymous";
      this.busy = false;
      this.router.navigate(['/']);
    }
  }

  promptConfirm(title: string, message: string, callback: () => void) {
    this.confirmTitle = title;
    this.confirmMessage = message;
    this.confirmCallback = callback;
    this.showConfirmDialog = true;
  }

  confirmAction() {
    if (this.confirmCallback) {
      this.confirmCallback();
    }
    this.showConfirmDialog = false;
    this.confirmCallback = null;
  }

  cancelConfirm() {
    this.showConfirmDialog = false;
    this.confirmCallback = null;
  }
}