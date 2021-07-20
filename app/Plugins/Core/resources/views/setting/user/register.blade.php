<div class="card tab-pane active show">
    <div class="card card-body">
        <div class="mb-3">
            <label class="form-label">注册默认用户组</label>
            <select class="form-select" v-model="data.core_user_reg_defuc">
            @foreach(plugins_core_user_reg_defuc() as $value)
                    <option value="{{$value->id}}">{{$value->name}}</option>
                @endforeach
            </select>
            <small>默认选择id为1的用户组</small>
        </div>

    </div>
</div>

