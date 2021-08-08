@extends("app")

@section('title',"帖子标签 - 新增")

@section('content')
    <div class="col-md-12">
        <div class="card card-body" id="vue-topic-tag-create">
            <h3 class="card-title">新增标签</h3>
            <form method="post" action="/admin/topic/tag/create?Redirect=admin/topic/tag/create" @@submit.prevent="submit" enctype="multipart/form-data">
                <x-csrf/>
                <div class="mb-3">
                    <label class="form-label">
                        名称
                    </label>
                    <input type="text" class="form-control" name="name" v-model="name" required>
                </div>
                <div class="mb-3">
                    <label class="form-label">
                        颜色
                    </label>
                    <input type="color" class="form-control form-control-color" name="color" v-model="color" required>
                </div>
                <div class="mb-3">
                    <label class="form-label">
                        图标
                    </label>
                    <input type="file" class="form-control" name="icon" v-model="icon" required>
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
