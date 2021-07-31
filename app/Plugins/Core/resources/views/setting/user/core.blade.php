<div class="card tab-pane active show">
    <div class="card card-body">
        <div class="mb-3">
            <label class="form-label">默认头像源
            </label>
            <select class="form-select" v-model="data.core_user_def_avatar">
                <option value="gavatar">Gavatar</option>
                <option value="multiavatar">Multiavatar</option>
            </select>
            <small>默认使用Gravatar的头像</small>
        </div>

        <div class="mb-3">
            <label class="form-label">Multiavatar头像演示
                <div>
                    <span class="avatar" style="background-image:url(/plugins/Core/image/Multiavatar.gif)"></span>
                </div>
            </label>
        </div>

        <div class="mb-3">
            <label class="form-label">头像缓存
            </label>
            <select class="form-select" v-model="data.core_user_avatar_cache">
                <option value="1">开启</option>
                <option value="2">关闭</option>
            </select>
            <small>默认开启</small>
        </div>

        <div class="mb-3">
            <label class="form-label">头像缓存时间
            </label>
            <input type="number" class="form-control" v-model="data.core_user_def_avatar_cache">
            <small>头像缓存时间/秒,减少查询,大幅度优化速度 默认10分钟</small>
        </div>

    </div>
</div>

