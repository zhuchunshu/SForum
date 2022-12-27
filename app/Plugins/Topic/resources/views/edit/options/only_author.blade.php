
<div class="col-12">
    <div class="card">
        <div class="card-body">
            <label class="form-check">
                <input type="checkbox" name="options[only_author]" class="form-check-input" @if(@$data->post->options->only_author){{("checked")}}@endif />
                <span class="form-check-label">评论仅作者可见</span>
            </label>
        </div>
    </div>
</div>