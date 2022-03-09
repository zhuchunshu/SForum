<div class="card card-body">
    <div class="mb-3">
        <label class="form-label">标题最少要求字数</label>
        <input type="text" class="form-control" v-model="data.topic_create_title_min">
        <small>当前: {{get_options("topic_create_title_min",1)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">标题最多要求字数</label>
        <input type="text" class="form-control" v-model="data.topic_create_title_max">
        <small>当前: {{get_options("topic_create_title_max",200)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">正文内容最少要求字数</label>
        <input type="text" class="form-control" v-model="data.topic_create_content_min">
        <small>当前: {{get_options("topic_create_content_min",10)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">发帖间隔时间/秒</label>
        <input type="number" class="form-control" v-model="data.topic_create_time">
        <small>当前: {{get_options("topic_create_time",120)}} 秒,防刷帖 建议不要低于一分钟</small>
    </div>

    <div class="mb-3">
        <label class="form-label">每页显示帖子数量</label>
        <input type="number" class="form-control" v-model="data.topic_home_num">
        <small>当前: {{get_options("topic_home_num",15)}}</small>
    </div>


</div>