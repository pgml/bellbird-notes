const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const vaxis = b.dependency("vaxis", .{ .target = target, .optimize = optimize });
    const dmp = b.dependency("diffmatchpatch", .{});
    //const zig_time_dep = b.dependency("zig-time", .{});
    const known_folders = b.dependency("known-folders", .{}).module("known-folders");
    const microwave = b.dependency("microwave", .{}).module("microwave");

    const exe = b.addExecutable(.{
        .name = "bbnotes",
        .root_module = b.createModule(.{
            .root_source_file = b.path("src/main.zig"),
            .target = target,
            .optimize = optimize,
            .imports = &.{},
        }),
    });

    exe.root_module.addImport("vaxis", vaxis.module("vaxis"));
    //exe.root_module.addImport("zig-time", zig_time_dep.module("zig-time"));
    exe.root_module.addImport("diffmatchpatch", dmp.module("diffmatchpatch"));
    exe.root_module.addImport("known-folders", known_folders);
    exe.root_module.addImport("microwave", microwave);

    b.installArtifact(exe);

    const run_step = b.step("run", "Run the app");
    const run_cmd = b.addRunArtifact(exe);
    run_step.dependOn(&run_cmd.step);

    run_cmd.step.dependOn(b.getInstallStep());

    if (b.args) |args| {
        run_cmd.addArgs(args);
    }

    const exe_tests = b.addTest(.{
        .root_module = exe.root_module,
    });

    const run_exe_tests = b.addRunArtifact(exe_tests);

    const test_step = b.step("test", "Run tests");
    test_step.dependOn(&run_exe_tests.step);
}
