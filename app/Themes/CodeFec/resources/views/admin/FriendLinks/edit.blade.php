@extends('app')

@section('title','修改友链')

@section('content')
    <div class="row row-cards">
        <div class="col-md-12">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">修改友链</h3>
                </div>
                <div class="card-body">
                    <form action="/admin/setting/friend_links" method="POST">
                        <x-csrf/>
                        <input type="hidden" name="id" value="{{$data->id}}">
                        <div class="row row-cards">
                            <div class="col-lg-4">
                                <label for="" class="form-label required">ID</label>
                                <input type="text" class="form-control" value="{{$data->id}}" required disabled>
                            </div>
                            <div class="col-lg-4">
                                <label for="" class="form-label required">名称</label>
                                <input type="text" name="name" class="form-control" value="{{$data->name}}" required>
                            </div>
                            <div class="col-lg-4">
                                <label for="" class="form-label required">链接</label>
                                <input type="text" name="link" value="{{$data->link}}" class="form-control" required>
                            </div>
                            <div class="col-lg-4">
                                <label for="" class="form-label">图标链接</label>
                                <input type="text" name="icon" value="{{$data->icon}}" class="form-control">
                            </div>
                            <div class="col-lg-4">
                                <label for="" class="form-label required">排序</label>
                                <input type="number" name="to_sort" class="form-control" value="{{$data->to_sort}}" required>
                                <small>数值越大越靠前</small>
                            </div>
                            <div class="col-lg-4">
                                <label for="" class="form-label required">新标签打开</label>
                                <select name="_blank" class="form-select">
                                    <option value="开启">开启</option>
                                    <option value="关闭" @if($data->_blank!='1') selected @endif>关闭</option>
                                </select>
                            </div>
                            <div class="col-12 row">
                                <div class="col"></div>
                                <div class="col-auto">
                                    <button type="submit" name="action" value="delete" class="btn btn-danger">删除</button>
                                    <button type="submit" class="btn btn-primary">提交</button>
                                </div>
                            </div>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </div>
@endsection

@section('headerBtn')
    <a class="btn btn-primary" href="/admin/setting/friend_links">友链列表</a>
@endsection