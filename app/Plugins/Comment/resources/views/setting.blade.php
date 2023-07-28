<div class="card card-body">
    <div class="row">
        <div class="mb-3 col-lg-3">
            <label class="form-label">评论最少字数</label>
            <input type="text" class="form-control" v-model="data.comment_create_min">
            <small>当前: {{get_options("comment_create_min",1)}}</small>
        </div>

        <div class="mb-3 col-lg-3">
            <label class="form-label">评论最多字数</label>
            <input type="text" class="form-control" v-model="data.comment_create_max">
            <small>当前: {{get_options("comment_create_max",200)}}</small>
        </div>

        <div class="mb-3 col-lg-3">
            <label class="form-label">回复最少字数</label>
            <input type="text" class="form-control" v-model="data.comment_reply_min">
            <small>当前: {{get_options("comment_reply_min",1)}}</small>
        </div>

        <div class="mb-3 col-lg-3">
            <label class="form-label">回复最多字数</label>
            <input type="text" class="form-control" v-model="data.comment_reply_max">
            <small>当前: {{get_options("comment_reply_max",200)}}</small>
        </div>

        <div class="mb-3 col-lg-3">
            <label class="form-label">评论间隔时间/秒</label>
            <input type="number" class="form-control" v-model="data.comment_create_time">
            <small>当前: {{get_options("comment_create_time",60)}} 秒,防灌水 建议不要低于一分钟</small>
        </div>


        <div class="mb-3 col-lg-3">
            <label class="form-label">每次加载多少条评论?</label>
            <input type="number" class="form-control" v-model="data.comment_page_count">
            <small>当前: {{get_options("comment_page_count",15)}}</small>
        </div>

        <div class="mb-3 col-lg-3">
            <label class="form-label">显示评论作者ip归属地</label>
            <select type="number" class="form-control" v-model="data.comment_author_ip">
                <option value="开启">开启</option>
                <option value="关闭">关闭</option>
            </select>
            <small>默认开启</small>
        </div>

        <div class="mb-3 col-lg-3">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.comment_show_desc">
                <span class="form-check-label">评论倒序显示</span>
            </label>
        </div>
        <div class="mb-3 col-lg-4">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.comment_emoji_close">
                <span class="form-check-label">关闭插入表情功能</span>
            </label>
        </div>

        <div class="mb-3 col-lg-4">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.comment_change_show">
                <span class="form-check-label">开启显示评论修改记录</span>
            </label>
        </div>

        <div class="mb-3 col-lg-4">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.comment_change_limit">
                <span class="form-check-label">开启评论修改时间限制</span>
            </label>
        </div>

        <div class="mb-3 col-lg-3">
            <label for="" class="form-label">
                评论修改时间限制 /分钟
            </label>
            <input type="number" class="form-control" v-model="data.comment_change_limit_time">
            <small>超过则不让修改评论，默认为5分钟</small>
        </div>

    </div>


</div>