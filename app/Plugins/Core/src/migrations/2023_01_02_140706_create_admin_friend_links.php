<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateAdminFriendLinks extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('friend_links', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->string('name');
            $table->text('link');
            $table->text('icon')->nullable();
            $table->integer('to_sort')->default(1);
            $table->boolean('_blank')->default(1);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('friend_links');
    }
}
