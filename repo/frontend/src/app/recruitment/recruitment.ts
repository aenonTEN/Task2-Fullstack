import { Component, OnInit } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { ApiService } from "../api.service";

@Component({
  selector: "app-recruitment",
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="section">
      <div class="section-header">
        <h2 class="section-title">Recruitment</h2>
      </div>
      
      <div class="glass-panel">
        <h3>Create Candidate</h3>
        <div class="grid">
          <label><span>Name</span><input [(ngModel)]="candName" /></label>
          <label><span>Phone</span><input [(ngModel)]="candPhone" /></label>
          <label><span>ID Number</span><input [(ngModel)]="candIdNumber" /></label>
          <label><span>Education</span><input [(ngModel)]="candEducation" /></label>
          <label><span>Experience</span><input [(ngModel)]="candExperienceYears" type="number" /></label>
          <label style="grid-column: 1 / -1;"><span>Skills</span><input [(ngModel)]="candSkills" placeholder="comma-separated" /></label>
        </div>
        <div class="row">
          <button (click)="createCandidate()" [disabled]="loading">Create</button>
          <button class="btn-secondary" (click)="loadCandidates()" [disabled]="loading">Refresh</button>
        </div>
      </div>

      <div class="glass-panel">
        <h3>Bulk Import</h3>
        <div class="row">
          <input type="file" (change)="onBulkFileSelected($event)" accept=".json" />
          <button (click)="bulkImport()" [disabled]="!bulkFile || bulkImporting">
            {{ bulkImporting ? 'Importing...' : 'Import' }}
          </button>
        </div>
        <div *ngIf="bulkProgress" class="loading">
          <span class="spinner"></span>{{ bulkProgress }}
        </div>
      </div>

      <div class="glass-panel">
        <div class="section-header">
          <h3>Candidates</h3>
          <span class="badge">{{ candidates.length }}</span>
        </div>
        
        <div *ngIf="loading" class="loading">
          <span class="spinner"></span>Loading...
        </div>
        
        <div *ngIf="!loading && candidates.length === 0" class="empty-state">
          <div class="empty-state-icon">📋</div>
          <div class="empty-state-title">No candidates</div>
        </div>
        
        <div *ngIf="!loading && candidates.length > 0">
          <table class="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Phone</th>
                <th>Education</th>
                <th>Experience</th>
                <th>Skills</th>
              </tr>
            </thead>
            <tbody>
              <tr *ngFor="let c of candidates">
                <td>{{ c.name }}</td>
                <td>{{ c.phone || '-' }}</td>
                <td>{{ c.education || '-' }}</td>
                <td>{{ c.experienceYears ? c.experienceYears + ' yrs' : '-' }}</td>
                <td>
                  <span *ngIf="c.skills?.length" class="tag-list">
                    <span *ngFor="let s of c.skills" class="tag">{{ s }}</span>
                  </span>
                  <span *ngIf="!c.skills?.length">-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="glass-panel">
        <h3>Search</h3>
        <div class="grid">
          <label><span>Keyword</span><input [(ngModel)]="searchKeyword" /></label>
          <label><span>Skill</span><input [(ngModel)]="searchSkill" /></label>
          <label><span>Education</span><input [(ngModel)]="searchEducation" /></label>
        </div>
        <div class="row">
          <button (click)="searchCandidates()" [disabled]="loading">Search</button>
        </div>
        
        <div *ngIf="searchResults.length > 0" class="card-grid" style="margin-top: 1rem;">
          <div *ngFor="let c of searchResults" class="card">
            <div class="card-header">
              <span class="card-title">{{ c.name }}</span>
              <span class="badge">{{ c.score || 0 }} pts</span>
            </div>
            <div class="card-body">
              <p>{{ c.matchReason }}</p>
            </div>
          </div>
        </div>
        
        <div *ngIf="searched && searchResults.length === 0" class="empty-state">
          <div class="empty-state-title">No results</div>
        </div>
      </div>
    </div>
  `
})
export class RecruitmentComponent implements OnInit {
  candidates: any[] = [];
  candName = "";
  candPhone = "";
  candIdNumber = "";
  candEducation = "";
  candExperienceYears: number | null = null;
  candSkills = "";
  searchKeyword = "";
  searchSkill = "";
  searchEducation = "";
  searchResults: any[] = [];
  searched = false;
  bulkFile: File | null = null;
  bulkImporting = false;
  bulkProgress = "";
  bulkResult: any = null;
  loading = false;
  searchLoading = false;

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.loadCandidates();
  }

  async loadCandidates() {
    this.loading = true;
    try {
      this.candidates = await this.api.candidates();
    } catch (e) {
      console.error(e);
    }
    this.loading = false;
  }

  async createCandidate() {
    this.loading = true;
    try {
      await this.api.createCandidate({
        name: this.candName,
        phone: this.candPhone,
        idNumber: this.candIdNumber,
        education: this.candEducation,
        experienceYears: this.candExperienceYears,
        skills: this.candSkills.split(",").map((s: string) => s.trim()).filter((s: string) => s)
      });
      this.candName = "";
      this.candPhone = "";
      this.candIdNumber = "";
      this.candEducation = "";
      this.candExperienceYears = null;
      this.candSkills = "";
      await this.loadCandidates();
    } catch (e) {
      console.error(e);
    }
    this.loading = false;
  }

  onBulkFileSelected(event: Event) {
    const input = event.target as HTMLInputElement;
    this.bulkFile = input.files?.[0] || null;
  }

  async bulkImport() {
    if (!this.bulkFile) return;
    this.bulkImporting = true;
    this.bulkProgress = "Reading...";
    try {
      const text = await this.bulkFile.text();
      const candidates = JSON.parse(text);
      this.bulkProgress = "Importing...";
      this.bulkResult = await this.api.bulkImport(candidates);
      await this.loadCandidates();
    } catch (e: any) {
      console.error(e);
    }
    this.bulkImporting = false;
    this.bulkProgress = "";
  }

  async searchCandidates() {
    this.searchLoading = true;
    this.searched = true;
    try {
      this.searchResults = await this.api.searchCandidates({
        keyword: this.searchKeyword,
        skill: this.searchSkill,
        education: this.searchEducation
      });
    } catch (e) {
      console.error(e);
    }
    this.searchLoading = false;
  }
}