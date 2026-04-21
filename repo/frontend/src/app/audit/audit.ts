import { Component } from "@angular/core";
import { CommonModule } from "@angular/common";
import { ApiService } from "../api.service";

@Component({
  selector: "app-audit",
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="section">
      <h2>Audit Records</h2>
      <div class="row">
        <button (click)="loadAudit()">Refresh</button>
      </div>
      <pre class="message">{{ auditRecords | json }}</pre>
    </div>
  `
})
export class AuditComponent {
  auditRecords: any[] = [];

  constructor(private api: ApiService) {}

  async loadAudit() {
    this.auditRecords = await this.api.auditRecords();
  }
}