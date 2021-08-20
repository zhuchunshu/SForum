<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateUsersOptionsTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('users_options', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->longText("qianming")->comment('签名')->nullable();
            $table->string("qq")->comment("qq")->nullable();
            $table->string("wx")->comment("微信")->nullable();
            $table->string("website")->comment("网站")->nullable();
            $table->string("email")->comment("展示邮箱")->nullable();
            $table->longText("options")->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('users_options');
    }
}
