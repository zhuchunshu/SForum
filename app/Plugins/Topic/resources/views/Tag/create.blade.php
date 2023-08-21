@extends("app")

@section('title',"帖子板块 - 新增")

@section('content')
    <div class="col-md-12">
        <div class="card card-body" id="vue-topic-tag-create">
            <h3 class="card-title">新增板块</h3>
            <form method="post" action="/admin/topic/tag/create?Redirect=admin/topic/tag/create">
                <x-csrf/>
                <div class="mb-3">
                    <label class="form-label">
                        名称
                    </label>
                    <input type="text" class="form-control" name="name" v-model="name" required>
                </div>
                <div class="mb-3 row">
                    <div class="col-6">
                        <label class="form-label">
                            颜色
                        </label>
                        <input type="color" class="form-control form-control-color" name="color" v-model="color" required>
                    </div>
                </div>
                <div class="mb-3">
                    <label class="form-label">
                        图标
                    </label>
                    <a href="https://tabler-icons.io/">https://tabler-icons.io/</a>
                    <textarea name="icon" v-model="icon" rows="10" class="form-control" required></textarea>
                </div>
                <div class="mb-3">
                    <label class="form-label"> 哪个用户组可使用此板块? </label>
                    <div class="row">
                        @foreach($userClass as $value)
                            <div class="col-4">
                                <label class="form-check form-switch">
                                    <input name="userClass[]" value="{{$value['name']}}" class="form-check-input" type="checkbox">
                                    <span class="form-check-label">{{$value['name']}}</span>
                                </label>
                            </div>
                        @endforeach
                    </div>
                    <small style="color:red">不选择则所有用户组都可用此板块</small>
                </div>
                <div class="mb-3">
                    <label class="form-label">描述</label>
                    <textarea name="description" class="form-control" rows="4"></textarea>
                </div>
                <div class="mb-3">
                    <button class="btn btn-primary">提交</button>
                </div>
            </form>
        </div>
    </div>
@endsection


@section("scripts")
    <script src="{{mix('plugins/Topic/js/tag.js')}}"></script>
@endsection
