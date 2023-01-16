@extends('app')
@section('title','新增页头菜单')
@section('content')
<div class="row row-cards">
    <div class="col-12">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">新增页头菜单</h3>
            </div>
            <div class="card-body" id="vue-create-header-menu">
                <form action="/admin/setting/menu/create" method="post" @@submit.prevent="submit">
                    <x-csrf/>
                    <div class="row">
                        <div class="col-lg-4">
                            <label for="" class="form-label required">名称</label>
                            <input type="text" v-model="data.name" class="form-control" required>
                        </div>
                        <div class="col-lg-4">
                            <label for="" class="form-label required">链接</label>
                            <input type="text" v-model="data.url" class="form-control" required>
                        </div>
                        <div class="col-lg-4">
                            <label for="" class="form-label required">排序 <small class="text-muted">(数字越小越靠前)</small> </label>
                            <input type="number" v-model="data.sort" class="form-control" required>
                        </div>
                        <div class="col-lg-4">
                            <label for="" class="form-label">上级菜单ID </label>
                            <input type="number" v-model="data.parent_id"  class="form-control">
                        </div>
                        <div class="col-lg-12">
                            <label for="" class="form-label required">图标 <small class="text-muted">svg代码</small> </label>
                            <textarea v-model="data.icon" id="" rows="5" class="form-control"></textarea>
                            <small> <a href="https://tabler-icons.io/">https://tabler-icons.io/</a> </small>
                        </div>
                        <div class="col-12 mt-3">
                            <button class="btn btn-primary" type="submit">提交</button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>
@endsection
@section('headerBtn')
    <a href="/admin/setting/menu" class="btn btn-primary">列表</a>
@endsection
@section('scripts')
    <script>
        var form_data={
            parent_id:"{{request()->input('parent_id')}}"
        }
    </script>
    <script src="{{file_hash('js/admin/setting.js')}}"></script>
@endsection