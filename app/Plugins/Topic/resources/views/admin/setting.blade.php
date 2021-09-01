<div id="common" class="card tab-pane">
    <div class="card card-body">
        <div class="mb-3">
            <label class="form-label">标题最少要求字数</label>
            <input type="text" min="1" max="3" class="form-control" v-model="data.topic_create_title_min">
            <small>当前: {{get_options("topic_create_title_min",1)}}</small>
        </div>

        <div class="mb-3">
            <label class="form-label">标题最多要求字数</label>
            <input type="text" min="1" max="3" class="form-control" v-model="data.topic_create_title_max">
            <small>当前: {{get_options("topic_create_title_max",200)}}</small>
        </div>

        <div class="mb-3">
            <label class="form-label">正文内容最少要求字数</label>
            <input type="text" min="1" max="3" class="form-control" v-model="data.topic_create_content_min">
            <small>当前: {{get_options("topic_create_content_min",10)}}</small>
        </div>

    </div>
</div>

