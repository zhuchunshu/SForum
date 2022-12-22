
<div class="col-12">
    <div class="card">
        <div class="card-body">
            <label class="form-check">
                <input type="checkbox" name="options[disable_comment]" @if($data->post->options->disable_comment){{("checked")}}@endif class="form-check-input"/>
                <span class="form-check-label">关闭评论</span>
            </label>
        </div>
    </div>
</div>