import { Component, OnInit } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { RouterModule, Router } from "@angular/router";
import { ApiService } from "./api.service";

@Component({
  selector: "app-root",
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterModule
  ],
  template: `
    <main class="container">
      <header class="header">
        <h1>Integrated Platform</h1>
        <div class="meta">
          <span class="pill" [class.ok]="status === 'authenticated'" [class.bad]="status === 'error'">
            {{ status }}
          </span>
        </div>
      </header>

      <section *ngIf="status !== 'authenticated'" class="glass-panel">
        <h2>Login</h2>
        <div class="grid">
          <label>
            <span>Username</span>
            <input [(ngModel)]="username" autocomplete="username" />
          </label>
          <label>
            <span>Password</span>
            <input [(ngModel)]="password" type="password" autocomplete="current-password" />
          </label>
        </div>
        <div class="row">
          <button (click)="login()" [disabled]="busy">Sign in</button>
          <span class="hint">Default scaffold user: <code>admin / password123</code></span>
        </div>
        <pre *ngIf="message" class="message">{{ message }}</pre>
      </section>

      <section *ngIf="status === 'authenticated'" class="glass-panel">
        <h2>Session</h2>
        <p><strong>Expires</strong>: <code>{{ expiresAt }}</code></p>
        <div *ngIf="api.me" class="message">
          <div><strong>User</strong>: <code>{{ api.me.userId }}</code></div>
          <div><strong>Roles</strong>: <code>{{ api.me.roleIds.join(', ') }}</code></div>
          <div><strong>Scope</strong>: <code>{{ api.me.scope | json }}</code></div>
        </div>
        <div class="row">
          <button class="btn-secondary" (click)="logout()" [disabled]="busy">Logout</button>
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
    </main>
  `
})
export class AppComponent implements OnInit {
  username = "admin";
  password = "password123";

  status: "anonymous" | "authenticated" | "error" = "anonymous";
  busy = false;
  message = "";
  expiresAt = "";

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
      this.message = JSON.stringify(e.error || e.message, null, 2);
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
}