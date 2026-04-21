import { Component } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { ApiService } from "../api.service";

@Component({
  selector: "app-tags",
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="section">
      <h2>Tags</h2>
      
      <h3>Create Tag</h3>
      <div class="grid">
        <label><span>Name</span><input [(ngModel)]="newTagName" /></label>
        <label><span>Color</span><input [(ngModel)]="newTagColor" type="color" /></label>
        <label><span>Description</span><input [(ngModel)]="newTagDesc" /></label>
      </div>
      <div class="row">
        <button (click)="createTag()">Create</button>
        <button (click)="loadTags()">Refresh</button>
      </div>
      <p *ngIf="tagMessage" class="message">{{ tagMessage }}</p>

      <h3>Available Tags</h3>
      <div class="tags-list">
        <div *ngFor="let t of tags" class="tag-item" [style.border-left]="'4px solid ' + t.color">
          <span class="tag-name">{{ t.name }}</span>
          <span class="tag-desc">{{ t.description }}</span>
          <button (click)="deleteTag(t.id)">Delete</button>
        </div>
      </div>

      <h3>Assign Tags</h3>
      <div class="grid">
        <label><span>Entity Type</span>
          <select [(ngModel)]="assignEntityType">
            <option value="candidate">Candidate</option>
            <option value="case">Case</option>
          </select>
        </label>
        <label><span>Entity ID</span><input [(ngModel)]="assignEntityId" /></label>
        <label style="grid-column: 1 / -1;"><span>Select Tags</span>
          <div class="tag-checkboxes">
            <label *ngFor="let t of tags">
              <input type="checkbox" (change)="toggleTag(t.id, $event)" [checked]="selectedTagIds.includes(t.id)" />
              <span [style.color]="t.color">{{ t.name }}</span>
            </label>
          </div>
        </label>
      </div>
      <div class="row">
        <button (click)="assignTags()">Assign</button>
      </div>
      <div *ngIf="assignMessage" class="message">{{ assignMessage }}</div>
    </div>
  `
})
export class TagsComponent {
  tags: any[] = [];
  newTagName = "";
  newTagColor = "#6b7280";
  newTagDesc = "";
  tagMessage = "";
  assignEntityType = "candidate";
  assignEntityId = "";
  selectedTagIds: string[] = [];
  assignMessage = "";

  constructor(private api: ApiService) {}

  async loadTags() {
    this.tags = await this.api.tags();
  }

  async createTag() {
    if (!this.newTagName.trim()) return;
    const result = await this.api.createTag({
      name: this.newTagName,
      color: this.newTagColor,
      description: this.newTagDesc
    });
    this.tagMessage = "Created: " + result.name;
    this.newTagName = "";
    this.newTagColor = "#6b7280";
    this.newTagDesc = "";
    await this.loadTags();
  }

  async deleteTag(tagId: string) {
    await this.api.deleteTag(tagId);
    this.tagMessage = "Deleted";
    await this.loadTags();
  }

  toggleTag(tagId: string, event: Event) {
    const checked = (event.target as HTMLInputElement).checked;
    if (checked) {
      if (!this.selectedTagIds.includes(tagId)) {
        this.selectedTagIds.push(tagId);
      }
    } else {
      this.selectedTagIds = this.selectedTagIds.filter(id => id !== tagId);
    }
  }

  async assignTags() {
    if (!this.assignEntityId || this.selectedTagIds.length === 0) return;
    await this.api.assignTags(this.assignEntityType, this.assignEntityId, this.selectedTagIds);
    this.assignMessage = `Assigned ${this.selectedTagIds.length} tag(s) to ${this.assignEntityType} ${this.assignEntityId}`;
  }
}