@extends("plugins.Core.app")

@section('title',"创建用户组")


@section('content')


    <div class="p-6 card bordered">
        <div class="form-control">
            <label class="cursor-pointer label">
                <span class="label-text">Neutral</span>
                <div>
                    <input type="radio" name="opt" checked="checked" class="radio" value="">
                    <span class="radio-mark"></span>
                </div>
            </label>
        </div>
        <div class="form-control">
            <label class="cursor-pointer label">
                <span class="label-text">Primary</span>
                <div>
                    <input type="radio" name="opt" checked="checked" class="radio radio-primary" value="">
                    <span class="radio-mark"></span>
                </div>
            </label>
        </div>
        <div class="form-control">
            <label class="cursor-pointer label">
                <span class="label-text">Secondary</span>
                <div>
                    <input type="radio" name="opt" checked="checked" class="radio radio-secondary" value="">
                    <span class="radio-mark"></span>
                </div>
            </label>
        </div>
        <div class="form-control">
            <label class="cursor-pointer label">
                <span class="label-text">Accent</span>
                <div>
                    <input type="radio" name="opt" checked="checked" class="radio radio-accent" value="">
                    <span class="radio-mark"></span>
                </div>
            </label>
        </div>
        <div class="form-control">
            <label class="label">
                <span class="label-text">Disabled</span>
                <input type="radio" name="opt" checked="checked" value="" disabled="disabled" class="radio radio-accent">
                <span class="radio-mark"></span>
            </label>
        </div>
    </div>






@endsection
