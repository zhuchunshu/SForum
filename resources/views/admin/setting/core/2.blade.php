<div class="card-body">
    <div class="mb-3">
        <label class="form-label">限流器: QPS限制</label>
        <input v-model="env.rate_limit_create" type="number" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">限流器: 峰值</label>
        <input v-model="env.rate_limit_capacity" type="number" class="form-control">
    </div>
</div>