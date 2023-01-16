@extends('app')

@section('title','创建小部件')
@section('content')

    <style type="text/css" media="screen">
        #editor {
            width: 100%;
            min-height: 600px;
            font-size:20px;
            margin-bottom:10px;
        }
    </style>

    <div class="row row-cards">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">创建小部件</h3>
                <div class="card-actions">
                    <a href="/admin/hook/components"> 返回部件列表</a>
                </div>
            </div>
            <div class="card-body" id="vue-admin-hook-component-create">
                <form @@submit.prevent="submit">
                    <div class="mb-3">
                        <label for="" class="form-label">小部件名称</label>
                        <input type="text" v-model="name" class="form-control" placeholder="建议纯英语字母,可以使用符号.来进行分割">
                    </div>
                    <div id="editor">加载失败...</div>
                    <div class="mt-3 row">
                        <div class="col"></div>
                        <div class="col-auto">
                            <button type="submit" class="btn btn-primary">保存</button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>

@endsection
@section('scripts')
    <script type="text/javascript" src="/js/ace/ace.js"></script>
    <script type="text/javascript" src="/js/ace/ext-language_tools.js"></script>
    <script type="text/javascript">
        const create_action = "/admin/hook/components/create"
        const lang = "php_laravel_blade";
    </script>
    <script src="{{mix('js/admin/component.js')}}"></script>
@endsection
