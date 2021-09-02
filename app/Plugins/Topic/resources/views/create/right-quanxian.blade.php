<div class="col-md-12">
    <div class="card card-body">
        <h3 class="card-title">阅读权限</h3>
        <div class="mb-3">
            <label class="form-label">鉴权类型</label>
            <select class="form-select" v-model="options.hidden.type">
                <option value="close">不启用</option>
                <option value="user">指定用户可见</option>
                <option value="user_class">指定用户组可见</option>
                <option value="login">登陆可见</option>
            </select>
        </div>
        <div class="mb-3" v-if="options.hidden.type==='user'">
            <label for="" class="form-label">选择用户</label>
            <div class="row g-2">
                <div class="col">
                    <input v-model="options.hidden.user.selected" class="form-control" list="UserdatalistOptions" placeholder="Type to search..."/>
                    <datalist id="UserdatalistOptions">
                        @foreach(\App\Plugins\User\src\Models\User::query()->select('id','username')->get() as $value)
                            <option value="{{$value->username}}">
                        @endforeach
                    </datalist>
                </div>
                <div class="col-auto">
                    <button type="button" @@click="hidden_user_add" class="btn btn-blue">添加</button>
                </div>
            </div>
            <div class="mb-3" style="margin-top:5px">
                <label for="" class="form-label">已选择:</label>
                <div v-for="value in options.hidden.user.list">
                    <span>
                        <span class="badge bg-blue">@{{ value }}</span>
                        <span @@click="hidden_user_remove(value)" style="margin-left:-2px;border-radius:0px;-moz-border-radius:0px;" class="badge bg-red">x</span>
                    </span>
                    <br/>
                </div>
            </div>
        </div>
        <div class="mb-3" v-if="options.hidden.type==='user_class'">
            <select class="form-select" v-model="options.hidden.user_class" multiple>
                @foreach(\App\Plugins\User\src\Models\UserClass::query()->select('id','name')->get() as $value)
                    <option value="{{$value->id}}">{{$value->name}}</option>
                @endforeach
            </select>
            <div class="mb-3" style="margin-top:5px">
                <label for="" class="form-label">已选择:</label>
                <div v-for="value in options.hidden.user_class">
                    ID: @{{ value }}<br/>
                </div>
            </div>
        </div>
    </div>
</div>

